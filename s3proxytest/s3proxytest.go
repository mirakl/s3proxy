package s3proxytest

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	jsonlib "encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"s3proxy/backend"
	"testing"
	"time"
	"crypto/rand"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"sync"
)

var (
	log = logging.MustGetLogger("s3proxy")

	UrlExpiration      = 15 * time.Minute
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
		for i:= range messages {
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

func ServeHTTP(t *testing.T, r *gin.Engine, method string, url string, authorization string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, nil)
	assert.Nil(t, err)

	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	r.ServeHTTP(w, req)

	return w
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

		var i int = 1
		for i = 0; i < size*1024; i++ {
			_, err := rand.Read(randomBytes)
			assert.Nil(t, err)
			w.Write(randomBytes)
		}

		// Reset file pointer to the beginning of the file
		defer file.Seek(0, 0)

		err = w.Flush()
		assert.Nil(t, err)

		stat, err := file.Stat()
		fmt.Println(stat.Name())
		fmt.Println(stat.Size())
		assert.Nil(t, err)

		return file, stat.Size()
	}

	return file, 0
}

// Retrieve a field from a json object
// only from direct fields, no sub document
func getFieldFromJson(t *testing.T, json []byte, field string) string {
	if json != nil {
		var objmap map[string]interface{}

		err := jsonlib.Unmarshal(json, &objmap)
		assert.Nil(t, err)

		return objmap[field].(string)
	}

	return ""
}

// Wrapper for http.NewRequest
// httpMethod : GET, POST etc ...
// url : target url
// contentType : if u want to define one
// bodySize : if the http call has a body, the size of it
// body : io.Reader of the body
// returns the http status code, the reponse body and if any an error
func httpCall(t *testing.T, httpMethod string, url string, contentType string, apiKey string, bodySize int64, body io.Reader) (int, []byte) {

	req, err := http.NewRequest(httpMethod, url, nil)
	assert.Nil(t, err)

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

	statusCode, body := httpCall(t, http.MethodGet, endpoint, "", "", -1, nil)

	version := getFieldFromJson(t, body, "version")

	return statusCode, version

}

// Get a presigned url for upload or download depending on the HTTP Method
// method : HTTP method like POST or GET
// host : localhost:8080
// key : bucket/folder/file.txt
// apiKey : authorization key
// returns the http status, the url and if any an error
func getPresignedUrl(t *testing.T, method string, host string, key string, apiKey string) (int, string) {

	endpoint := "http://" + host + "/api/v1/presigned/url/" + key

	statusCode, body := httpCall(t, method, endpoint, "", apiKey, -1, nil)

	url := getFieldFromJson(t, body, "url")

	return statusCode, url
}

// Upload a random file from a s3 backend using a presigned url
// url : full presigned url
// apiKey : authorization key
// returns an http status code like 200, the uploaded file created and if any an error
func uploadFile(t *testing.T, url string, apiKey string) (statusCode int, uploadedFile *os.File) {

	uploadedFile, size := createTempFileInMB(t, 10) // 10MB

	statusCode, _ = httpCall(t, http.MethodPut, url, "binary/octet-stream", apiKey, size, uploadedFile)

	return statusCode, uploadedFile
}

// Download a file from a s3 backend using a presigned url
// url : full presigned url
// apiKey : authorization key
// returns an http status code like 200, the downloaded file and if any an error
func downloadFile(t *testing.T, url string, apiKey string) (int, *os.File) {

	file, _ := createTempFileInMB(t, 0)

	statusCode, binary := httpCall(t, http.MethodGet, url, "binary/octet-stream", apiKey, -1, nil)

	io.Copy(file, bytes.NewReader(binary))

	return statusCode, file
}

// Delete a file in a s3 backend
// host : localhost:8080
// key : bucket/folder/file.txt
// apiKey : authorization key
// returns an http status code like 200 and if any an error
func deleteFile(t *testing.T, host string, key string, apiKey string) int {

	endpoint := "http://" + host + "/api/v1/object/" + key

	statusCode, _ := httpCall(t, http.MethodDelete, endpoint, "", apiKey, -1, nil)

	return statusCode
}

// Wrapper for presigned URL
func getPresignedUrlForUpload(t *testing.T, host string, key string, apiKey string) (int, string) {
	return getPresignedUrl(t, http.MethodPost, host, key, apiKey)
}

// Wrapper for presigned URL
func getPresignedUrlForDownload(t *testing.T, host string, key string, apiKey string) (int, string) {
	return getPresignedUrl(t, http.MethodGet, host, key, apiKey)
}

// Used for integration and end-to-end test with the following scenario :
// 1- Get a presigned url for upload
// 1- Upload the file
// 2- Get a presgined url for download
// 2- Download the file uploaded in the step before
// 3- Check if the files are identical (checksum MD5)
// 4- Delete the file
// 5- Try to download the file again => should get 404
func RunSimpleScenarioForS3proxy(t *testing.T, s3proxyHost string) {

	key := "s3proxy-bucket/dummyfolder/dummyfile"

	// UPLOAD a temporary file to the s3 backend
	// create presigned url for uploading the file
	statusCode, uploadUrl := getPresignedUrlForUpload(t, s3proxyHost, key, ServerAPIKey)

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	// should have an url
	require.NotEqual(t, uploadUrl, "")

	statusCode, uploadedFile := uploadFile(t, uploadUrl, ServerAPIKey)
	//defer os.Remove(uploadedFile.Name())

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	// DOWNLOAD the file previously uploaded
	// create presigned url for downloading the file
	statusCode, downloadUrl := getPresignedUrlForDownload(t, s3proxyHost, key, ServerAPIKey)

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	// should have an url
	require.NotEqual(t, downloadUrl, "")

	statusCode, downloadedFile := downloadFile(t, downloadUrl, ServerAPIKey)
	//defer os.Remove(downloadedFile.Name())

	// should return 200
	require.Equal(t, http.StatusOK, statusCode)

	// Verify the files are the same
	require.True(t, verifyFileCheckSumEquality(t, uploadedFile, downloadedFile))

	// DELETE PART
	// Delete the file should return 200
	require.Equal(t, http.StatusOK, deleteFile(t, s3proxyHost, key, ServerAPIKey))

	// Last check, try to download again the file
	statusCode, redownloadedFile := downloadFile(t, downloadUrl, ServerAPIKey)
	defer os.Remove(redownloadedFile.Name())

	// should return 404 because the file has been deleted
	require.Equal(t, http.StatusNotFound, statusCode)
}
