package s3proxytest

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	jsonlib "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/mirakl/s3proxy/backend"
	logging "github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	log = logging.MustGetLogger("s3proxy")

	URLExpiration      = 15 * time.Minute
	ServerAPIKey       = "3f300bdc-0028-11e8-ba89-0ed5f89f718b"
	MinioBackendConfig = backend.S3BackendConfig{
		Host:             "minio:9000",
		Region:           "eu-west-1",
		AccessKey:        "AKIAIOSFODNN7EXAMPLE",
		SecretKey:        "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		DisableSSL:       true, // For minio : True
		S3ForcePathStyle: true, // Form minio : True
	}
)

// Wait for bucket creation and s3proxy readiness
func WaitForRessources(t *testing.T, s3proxyHost string) {

	messages := make(chan string)
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		WaitForBucket(t, "s3proxy-bucket", MinioBackendConfig)
		messages <- "bucket"
	}()

	go func() {
		defer wg.Done()
		WaitForS3proxy(t, s3proxyHost)
		messages <- "s3proxy"
	}()

	go func() {
		for i := range messages {
			log.Debug("Available ressource : %v", i)
		}
	}()

	wg.Wait()
}

// Wait for s3proxy readiness
func WaitForS3proxy(t *testing.T, s3proxyHost string) {

	for i := 0; i < 5; i++ {
		statusCode, _ := getHealthCheck(t, s3proxyHost)

		if statusCode == http.StatusOK {
			return
		}

		log.Debug("Waiting for s3proxy '%s' ...", s3proxyHost)

		time.Sleep(3 * time.Second)
	}

	assert.Fail(t, fmt.Sprintf("s3proxy '%s' not listening !", s3proxyHost))
}

// WaitForBucket is waiting for minio is up and bucket has been created
func WaitForBucket(t *testing.T, bucketName string, config backend.S3BackendConfig) {

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		Endpoint:         aws.String(config.Host),
		DisableSSL:       aws.Bool(config.DisableSSL),
		S3ForcePathStyle: aws.Bool(config.S3ForcePathStyle),
	}

	if config.Region != "" {
		s3Config.Region = aws.String(config.Region)
	}

	sess, err := session.NewSession(s3Config)
	assert.Nil(t, err)

	client := s3.New(sess)

	for i := 0; i < 5; i++ {
		result, err := client.ListBuckets(&s3.ListBucketsInput{})
		assert.Nil(t, err)

		for _, bucket := range result.Buckets {
			if aws.StringValue(bucket.Name) == bucketName {
				return
			}
		}

		log.Debug("Waiting for bucket '%s' ...", bucketName)

		time.Sleep(3 * time.Second)
	}

	assert.Fail(t, fmt.Sprintf("Bucket '%s' not found !", bucketName))
}

func ServeHTTPWithBody(t *testing.T, r *gin.Engine, method string, url string, body io.Reader, bodySize int, authorization string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()

	req, err := http.NewRequest(method, url, body)
	assert.Nil(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(bodySize))
	}

	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	r.ServeHTTP(w, req)

	return w
}

func ServeHTTP(t *testing.T, r *gin.Engine, method string, url string, authorization string) *httptest.ResponseRecorder {
	return ServeHTTPWithBody(t, r, method, url, nil, 0, authorization)
}

func ServeCreatePresignedURLForUpload(t *testing.T, r *gin.Engine, bucket string, key string, authorization string) *httptest.ResponseRecorder {
	return ServeHTTP(t, r, http.MethodPost, fmt.Sprintf("/api/v1/presigned/url/%v%v", bucket, key), authorization)
}

func ServeCreatePresignedURLForDownload(t *testing.T, r *gin.Engine, bucket string, key string, authorization string) *httptest.ResponseRecorder {
	return ServeHTTP(t, r, http.MethodGet, fmt.Sprintf("/api/v1/presigned/url/%v%v", bucket, key), authorization)
}

func ServeDeleteObject(t *testing.T, r *gin.Engine, bucket string, key string, authorization string) *httptest.ResponseRecorder {
	return ServeHTTP(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/object/%v%v", bucket, key), authorization)
}

func ServeBulkDeleteObject(t *testing.T, r *gin.Engine, bucket string, keys []string, authorization string) *httptest.ResponseRecorder {
	data := url.Values{}

	for _, v := range keys {
		data.Add("key", v)
	}

	return ServeHTTPWithBody(t, r, http.MethodPost, fmt.Sprintf("/api/v1/object/delete/%v", bucket), strings.NewReader(data.Encode()), len(data.Encode()), authorization)
}

func ServeCopyObject(t *testing.T, r *gin.Engine, sourceBucket string, sourceKey string, destinationBucket string, destinationKey string, authorization string) *httptest.ResponseRecorder {
	params := make(url.Values)

	if destinationBucket != "" {
		params.Set("destBucket", destinationBucket)
	}
	if destinationKey != "" {
		params.Set("destKey", destinationKey)
	}

	queryParams := ""

	if len(params) > 0 {
		queryParams = "?" + params.Encode()
	}

	return ServeHTTP(t, r, http.MethodPost, fmt.Sprintf("/api/v1/object/copy/%v%v%v", sourceBucket, sourceKey, queryParams), authorization)
}

func CatchPanic() {
	// if panic, recover first
	err := recover()

	// Print stacktrace in case of panic
	if err != nil {
		log.Fatal(err.(*errors.Error).ErrorStack())
	}
}

// return SHA256 of a file content
func getFileSHA256(t *testing.T, file *os.File) ([]byte, error) {
	h := sha256.New()

	_, err := io.Copy(h, file)
	assert.Nil(t, err)

	return h.Sum(nil), nil
}

// verify if two files has the same checksum
func verifyFileCheckSumEquality(t *testing.T, file1 *os.File, file2 *os.File) bool {

	md1, err := getFileSHA256(t, file1)
	assert.Nil(t, err)

	md2, err := getFileSHA256(t, file2)
	assert.Nil(t, err)

	return string(md1) == string(md2)
}

// create a temporary file
// if size = 0 => empty file otherwise a file with random characters
func createTempFileInMB(t *testing.T, size int) (*os.File, int64) {

	file, err := ioutil.TempFile(os.TempDir(), "s3proxy")
	assert.Nil(t, err)

	if size > 0 {
		w := bufio.NewWriter(file)

		randomBytes := make([]byte, 1024)

		for i := 0; i < size*1024; i++ {
			_, err := rand.Read(randomBytes)
			assert.Nil(t, err)
			_, err = w.Write(randomBytes)
			assert.Nil(t, err)
		}

		// Reset file pointer to the beginning of the file
		defer func() {
			_, err := file.Seek(0, 0)
			assert.Nil(t, err)
		}()

		err = w.Flush()
		assert.Nil(t, err)

		stat, err := file.Stat()
		assert.Nil(t, err)

		return file, stat.Size()
	}

	return file, 0
}

// Retrieve a field from a json object
// only from direct fields, no sub document
func getFieldFromJSON(t *testing.T, json []byte, field string) string {
	if json != nil {
		var objmap map[string]interface{}

		err := jsonlib.Unmarshal(json, &objmap)
		assert.Nil(t, err)

		if val, ok := objmap[field]; ok {
			return val.(string)
		}
	}

	return ""
}

// Wrapper for http.NewRequest
// httpMethod : GET, POST etc ...
// url : target url
// contentType : if u want to define one
// bodySize : if the http call has a body, the size of it
// body : io.Reader of the body
// expiration : link expiration duration
// returns the http status code, the reponse body and if any an error
func httpCall(t *testing.T, httpMethod string, url string, contentType string, apiKey string, bodySize int64, body io.Reader, expiration string) (int, []byte) {
	req, err := http.NewRequest(httpMethod, url, nil)
	assert.Nil(t, err)

	if expiration != "" {
		q := req.URL.Query()
		q.Add("expiration", expiration)
		req.URL.RawQuery = q.Encode()
	}

	if apiKey != "" {
		req.Header.Add("Authorization", apiKey)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if body != nil {
		req.Body = ioutil.NopCloser(body)
		req.ContentLength = bodySize
	}

	httpClient := &http.Client{
		Timeout: time.Second * 20,
	}

	response, err := httpClient.Do(req)
	assert.Nil(t, err)

	var statusCode int
	var bytes []byte

	if response != nil {
		defer response.Body.Close()

		statusCode = response.StatusCode

		bytes, err = ioutil.ReadAll(response.Body)
		assert.Nil(t, err)
	}

	return statusCode, bytes
}

// Call health check on s3proxy
func getHealthCheck(t *testing.T, host string) (int, string) {
	endpoint := "http://" + host + "/"

	statusCode, body := httpCall(t, http.MethodGet, endpoint, "", "", -1, nil, "")

	version := getFieldFromJSON(t, body, "version")

	return statusCode, version

}

// Get a presigned url for upload or download depending on the HTTP Method
// method : HTTP method like POST or GET
// host : localhost:8080
// key : bucket/folder/file.txt
// apiKey : authorization key
// expiration : link expiration duration
// returns the http status, the url and if any an error
func getPresignedURL(t *testing.T, method string, host string, key string, apiKey string, expiration string) (int, string) {

	endpoint := "http://" + host + "/api/v1/presigned/url/" + key

	statusCode, body := httpCall(t, method, endpoint, "", apiKey, -1, nil, expiration)

	url := getFieldFromJSON(t, body, "url")

	return statusCode, url
}

// Upload a random file from a s3 backend using a presigned url
// url : full presigned url
// apiKey : authorization key
// returns an http status code like 200, the uploaded file created and if any an error
func uploadFile(t *testing.T, url string, apiKey string) (int, *os.File) {

	uploadedFile, size := createTempFileInMB(t, 10) // 10MB

	statusCode, _ := httpCall(t, http.MethodPut, url, "binary/octet-stream", apiKey, size, uploadedFile, "")

	return statusCode, uploadedFile
}

// Download a file from a s3 backend using a presigned url
// url : full presigned url
// apiKey : authorization key
// returns an http status code like 200, the downloaded file and if any an error
func downloadFile(t *testing.T, url string, apiKey string) (int, *os.File) {

	file, _ := createTempFileInMB(t, 0)

	statusCode, binary := httpCall(t, http.MethodGet, url, "binary/octet-stream", apiKey, -1, nil, "")

	_, err := io.Copy(file, bytes.NewReader(binary))
	assert.Nil(t, err)

	return statusCode, file
}

// Delete a file in a s3 backend
// host : localhost:8080
// key : bucket/folder/file.txt
// apiKey : authorization key
// returns an http status code like 200 and if any an error
func deleteFile(t *testing.T, host string, key string, apiKey string) int {

	endpoint := "http://" + host + "/api/v1/object/" + key

	statusCode, _ := httpCall(t, http.MethodDelete, endpoint, "", apiKey, -1, nil, "")

	return statusCode
}

func deleteBatchFile(t *testing.T, host string, bucket string, keys []string, apiKey string) int {

	data := url.Values{}

	for _, v := range keys {
		data.Add("key", v)
	}

	endpoint := "http://" + host + "/api/v1/object/delete/" + bucket

	statusCode, _ := httpCall(t, http.MethodPost, endpoint, "application/x-www-form-urlencoded", apiKey, int64(len(data.Encode())), strings.NewReader(data.Encode()), "")

	return statusCode
}

// Copy file
func copyFile(t *testing.T, host string, sourceBucket string, sourceKey string, destinationBucket string, destinationKey string, apiKey string) (int, string) {

	params := make(url.Values)

	if destinationBucket != "" {
		params.Set("destBucket", destinationBucket)
	}
	if destinationKey != "" {
		params.Set("destKey", destinationKey)
	}

	queryParams := ""

	if len(params) > 0 {
		queryParams = "?" + params.Encode()
	}

	endpoint := fmt.Sprintf("http://%v/api/v1/object/copy/%v%v%v", host, sourceBucket, sourceKey, queryParams)

	statusCode, body := httpCall(t, http.MethodPost, endpoint, "", apiKey, -1, nil, "")

	msg := getFieldFromJSON(t, body, "error")

	return statusCode, msg
}

// Wrapper for presigned URL
func getPresignedURLForUpload(t *testing.T, host string, key string, apiKey string, expiration string) (int, string) {
	return getPresignedURL(t, http.MethodPost, host, key, apiKey, expiration)
}

// Wrapper for presigned URL
func getPresignedURLForDownload(t *testing.T, host string, key string, apiKey string, expiration string) (int, string) {
	return getPresignedURL(t, http.MethodGet, host, key, apiKey, expiration)
}

// checkUpload checks upload scenario : get presigned url + upload a file
func checkUpload(t *testing.T, s3proxyHost string, fullKey string, expiration string) *os.File {
	// create presigned url for uploading the file
	statusCode, uploadURL := getPresignedURLForUpload(t, s3proxyHost, fullKey, ServerAPIKey, expiration)

	checkExpiration(t, uploadURL, expiration)

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	// should have an url
	require.NotEqual(t, uploadURL, "")

	statusCode, uploadedFile := uploadFile(t, uploadURL, ServerAPIKey)
	//defer os.Remove(uploadedFile.Name())

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	return uploadedFile
}

// checkDownload checks download scenario : get presigned url + download the file
func checkDownload(t *testing.T, s3proxyHost string, fullKey string, statusCodeToCheck int, expiration string) *os.File {
	// create presigned url for downloading the file
	statusCode, downloadURL := getPresignedURLForDownload(t, s3proxyHost, fullKey, ServerAPIKey, expiration)

	checkExpiration(t, downloadURL, expiration)

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	// should have an url
	require.NotEqual(t, downloadURL, "")

	statusCode, downloadedFile := downloadFile(t, downloadURL, ServerAPIKey)
	//defer os.Remove(downloadedFile.Name())

	// should return 200
	require.Equal(t, statusCodeToCheck, statusCode)

	return downloadedFile
}

func checkExpiration(t *testing.T, uploadURL string, expiration string) {

	parsedURL, err := url.Parse(uploadURL)
	assert.NoError(t, err)
	expiresFromURLParam := parsedURL.Query().Get("X-Amz-Expires")
	expiresFromURL, err := strconv.ParseInt(expiresFromURLParam, 10, 64)
	assert.NoError(t, err)

	if expiration != "" {
		duration, err := time.ParseDuration(expiration)
		assert.Nil(t, err)

		expectedExpiration := int64(duration.Seconds())
		require.Equal(t, expectedExpiration, expiresFromURL)

	} else {
		// default duration is 15m, so 900s
		require.Equal(t, int64(900), expiresFromURL)
	}
}

// checkDownload checks copy scenario
func checkCopy(t *testing.T, s3proxyHost string, sourceBucket string, sourceKey string, destBucket string, destKey string) {
	// copy the file
	statusCode, message := copyFile(t, s3proxyHost, sourceBucket, sourceKey, destBucket, destKey, ServerAPIKey)

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	// should be empty, no error
	require.Empty(t, message)

	sourceFile := checkUpload(t, s3proxyHost, sourceBucket+sourceKey, "")
	defer os.Remove(sourceFile.Name())

	destinationFile := checkDownload(t, s3proxyHost, destBucket+destKey, http.StatusOK, "")
	defer os.Remove(destinationFile.Name())

	// Verify the files are the same
	require.True(t, verifyFileCheckSumEquality(t, sourceFile, destinationFile))
}

// checkDelete checks delete scenario : delete the file + try to download again
func checkDelete(t *testing.T, s3proxyHost string, fullKey string) {
	// Delete the file should return 200
	require.Equal(t, http.StatusOK, deleteFile(t, s3proxyHost, fullKey, ServerAPIKey))

	// Last check, try to download again the file
	// should return 404 because the file has been deleted
	checkDownload(t, s3proxyHost, fullKey, http.StatusNotFound, "")
}

// checkDelete checks delete scenario : delete the file + try to download again => should return 404
func checkBatchDelete(t *testing.T, s3proxyHost string, bucket string, keys []string) {
	// Delete the files should return 200
	require.Equal(t, http.StatusOK, deleteBatchFile(t, s3proxyHost, bucket, keys, ServerAPIKey))

	for _, key := range keys {
		// Last check, try to download again the file
		// should return 404 because the file has been deleted
		file := checkDownload(t, s3proxyHost, bucket+key, http.StatusNotFound, "")
		defer os.Remove(file.Name())
	}
}

// Used for integration and end-to-end test with the following scenario :
// 1- Get a presigned url for upload
// 1- Upload the file
// 2- Get a presgined url for download
// 2- Download the file uploaded in the step before
// 3- Check if the files are identical (checksum MD5)
// 4- Delete the file
// 5- Try to download the file again => should get 404
// 6- Batch delete files
func RunSimpleScenarioForS3proxy(t *testing.T, s3proxyHost string) {
	bucket := "s3proxy-bucket"
	key := "/dummyfolder/dummyfile"
	fullKey := bucket + key

	// UPLOAD a temporary file to the s3 backend
	uploadedFile := checkUpload(t, s3proxyHost, fullKey, "")
	defer os.Remove(uploadedFile.Name())

	// DOWNLOAD the file previously uploaded
	downloadedFile := checkDownload(t, s3proxyHost, fullKey, http.StatusOK, "")
	defer os.Remove(downloadedFile.Name())

	// Verify the files are the same
	require.True(t, verifyFileCheckSumEquality(t, uploadedFile, downloadedFile))

	// UPLOAD a temporary file to the s3 backend with expiration
	getUploadFileWithExpiration := checkUpload(t, s3proxyHost, fullKey, "25m")
	defer os.Remove(getUploadFileWithExpiration.Name())

	// DOWNLOAD the file previously uploaded with expiration
	getDownloadFileWithExpiration := checkDownload(t, s3proxyHost, fullKey, http.StatusOK, "25m")
	defer os.Remove(getDownloadFileWithExpiration.Name())

	// Verify the expiration param is taken into account
	require.True(t, verifyFileCheckSumEquality(t, getUploadFileWithExpiration, getDownloadFileWithExpiration))

	// COPY objects
	checkCopy(t, s3proxyHost, bucket, key, bucket, key+"2")

	copiedFile := checkDownload(t, s3proxyHost, fullKey+"2", http.StatusOK, "")
	defer os.Remove(copiedFile.Name())

	// DELETE object
	checkDelete(t, s3proxyHost, fullKey+"2")

	// COPY objects for bulk delete check
	checkCopy(t, s3proxyHost, bucket, key, bucket, key+"3")
	checkCopy(t, s3proxyHost, bucket, key, bucket, key+"4")
	checkCopy(t, s3proxyHost, bucket, key, bucket, key+"5")

	keys := []string{key + "3", key + "4", key + "5"}
	checkBatchDelete(t, s3proxyHost, bucket, keys)
}
