package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mirakl/s3proxy/backend/s3backend"

	"github.com/gin-gonic/gin"
	"github.com/mirakl/s3proxy/backend"
	"github.com/mirakl/s3proxy/backend/backendtest"
	"github.com/mirakl/s3proxy/router"
	"github.com/mirakl/s3proxy/s3proxytest"
	"github.com/stretchr/testify/assert"
)

var (
	s3Backend backend.Backend
	r         *gin.Engine

	expiration     = 15 * time.Minute
	dummyBucket    = "dummybucket"
	dummyFile      = "/dummyfolder/dummyfile"
	s3proxyVersion = "9.9.9"
	awsRegion      = "eu-west-1"
	accessKey      = "123456"
	secretKey      = "ABCDEFGH12345"
	serverAPIKey   = "ABCD-123"
)

// Launch gin server with a fake backend implementation
func setup() {
	var err error

	s3Backend, err = backendtest.New(s3backend.Config{
		Region:    awsRegion,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		os.Exit(1)
	}

	r = router.NewGinEngine(gin.TestMode, s3proxyVersion, expiration, "", s3Backend)
}

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	os.Exit(retCode)
}

func unmarshallJSON(t *testing.T, bytes []byte) map[string]interface{} {
	var objmap map[string]interface{}

	err := json.Unmarshal(bytes, &objmap)
	require.NoError(t, err)

	return objmap
}

// Check ping responding with the version number
func TestPing(t *testing.T) {
	w := s3proxytest.ServeHTTP(t, r, http.MethodGet, "/", "")
	assert.Equal(t, http.StatusOK, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	assert.Equal(t, objmap["version"], s3proxyVersion)
}

// Generate a presigned url for an upload
func TestCreateUrlForUploadOK(t *testing.T) {
	w := s3proxytest.ServeCreatePresignedURLForUpload(t, r, dummyBucket, dummyFile, "")
	assert.Equal(t, http.StatusOK, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	url := objmap["url"]
	assert.Contains(t, url, dummyBucket)
	assert.Contains(t, url, dummyFile)
	assert.Contains(t, url, awsRegion)
	assert.Contains(t, url, "X-Amz-Signature")
	assert.Contains(t, url, "X-Amz-Expires=900")
}

// Generate a presigned url for a download
func TestCreateUrlForDownloadOK(t *testing.T) {
	w := s3proxytest.ServeCreatePresignedURLForDownload(t, r, dummyBucket, dummyFile, "")
	assert.Equal(t, http.StatusOK, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	url := objmap["url"]
	assert.Contains(t, url, dummyFile)
	assert.Contains(t, url, awsRegion)
	assert.Contains(t, url, "X-Amz-Signature")
	assert.Contains(t, url, "X-Amz-Expires=900")
}

// Check the delete API, should always return 200 even if the object is not present
func TestDeleteOK(t *testing.T) {
	for i := 0; i < 2; i++ {
		w := s3proxytest.ServeDeleteObject(t, r, dummyBucket, dummyFile, "")
		assert.Equal(t, http.StatusOK, w.Code)

		objmap := unmarshallJSON(t, w.Body.Bytes())
		assert.Contains(t, objmap["response"], "ok")
	}
}

func TestBulkDeleteOK(t *testing.T) {
	w := s3proxytest.ServeBulkDeleteObject(t, r, dummyBucket, []string{"/toto/file1", "/toto/file2", "/toto/file2"}, "")
	assert.Equal(t, http.StatusOK, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	assert.Contains(t, objmap["response"], "ok")
}

func TestBulkDelete400BadRequest(t *testing.T) {
	w := s3proxytest.ServeBulkDeleteObject(t, r, dummyBucket, []string{}, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCopyOK(t *testing.T) {
	w := s3proxytest.ServeCopyObject(t, r, dummyBucket, dummyFile, dummyBucket, dummyFile+"2", "")
	assert.Equal(t, http.StatusOK, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	assert.Contains(t, objmap["response"], "ok")
}

func TestCopyMissingDestinationBucket(t *testing.T) {
	w := s3proxytest.ServeCopyObject(t, r, dummyBucket, dummyFile, "", dummyFile+"2", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	assert.Contains(t, objmap["error"], "Missing destination bucket")
}

func TestCopyMissingDestinationKey(t *testing.T) {
	w := s3proxytest.ServeCopyObject(t, r, dummyBucket, dummyFile, dummyBucket, "", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	assert.Contains(t, objmap["error"], "Missing destination key")
}

func TestCopyNoSuchBucket(t *testing.T) {
	w := s3proxytest.ServeCopyObject(t, r, "notfound", dummyFile, dummyBucket, dummyFile+"2", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	assert.Contains(t, objmap["error"], "No such bucket")

	w = s3proxytest.ServeCopyObject(t, r, dummyBucket, dummyFile, "notfound", dummyFile+"2", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	objmap = unmarshallJSON(t, w.Body.Bytes())
	assert.Contains(t, objmap["error"], "No such bucket")
}

func TestCopyNoSuchKey(t *testing.T) {
	w := s3proxytest.ServeCopyObject(t, r, dummyBucket, "/notfound", dummyBucket, dummyFile+"2", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	objmap := unmarshallJSON(t, w.Body.Bytes())
	assert.Contains(t, objmap["error"], "No such key")
}

// Check if we are getting a 500 when we have a panic. Should be handle by the recovery middleware
// we are using delete action which fires a fake panic when "error" is in the key
func TestRecoveryMiddleware(t *testing.T) {
	w := s3proxytest.ServeDeleteObject(t, r, dummyBucket, "/error", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Some 404 checks
func Test404(t *testing.T) {

	// missing elements in path
	w := s3proxytest.ServeHTTP(t, r, http.MethodGet, "/api/v1/presigned/url/dummybucket", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = s3proxytest.ServeHTTP(t, r, http.MethodGet, "/api/v1/presigned/url", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = s3proxytest.ServeHTTP(t, r, http.MethodGet, "/api/v1/presigned/url/coucou", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = s3proxytest.ServeHTTP(t, r, http.MethodPost, "/api/v1/presigned/coucou", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = s3proxytest.ServeHTTP(t, r, http.MethodDelete, "/api/v1/presigned/", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	// check unsupported method
	w = s3proxytest.ServeHTTP(t, r, http.MethodPut, "/api/v1/object/dummybucket/dummyfolder/dummyfile", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Check authorization with valid api key
func TestAuthorizationOK(t *testing.T) {

	// create a server with api key protection
	r = router.NewGinEngine(gin.ReleaseMode, s3proxyVersion, urlExpiration, serverAPIKey, s3Backend)

	// ping endpoint should not be protected
	w := s3proxytest.ServeHTTP(t, r, http.MethodGet, "/", "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Other endpoints should be protected
	w = s3proxytest.ServeCreatePresignedURLForUpload(t, r, dummyBucket, dummyFile, serverAPIKey)
	assert.Equal(t, http.StatusOK, w.Code)

	w = s3proxytest.ServeCreatePresignedURLForDownload(t, r, dummyBucket, dummyFile, serverAPIKey)
	assert.Equal(t, http.StatusOK, w.Code)

	w = s3proxytest.ServeDeleteObject(t, r, dummyBucket, dummyFile, serverAPIKey)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Check authorization verification with missing api key
func TestAuthorization401(t *testing.T) {

	// create a server with api key protection
	r = router.NewGinEngine(gin.ReleaseMode, s3proxyVersion, urlExpiration, serverAPIKey, s3Backend)

	// ping endpoint should not be protected
	w := s3proxytest.ServeHTTP(t, r, http.MethodGet, "/", "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Other endpoints should be protected
	w = s3proxytest.ServeCreatePresignedURLForUpload(t, r, dummyBucket, dummyFile, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = s3proxytest.ServeCreatePresignedURLForDownload(t, r, dummyBucket, dummyFile, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = s3proxytest.ServeDeleteObject(t, r, dummyBucket, dummyFile, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

}
