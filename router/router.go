package router

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/mirakl/s3proxy/backend"
	"github.com/mirakl/s3proxy/middleware"
	"github.com/op/go-logging"
	"net/http"
	"time"
)

var (
	log = logging.MustGetLogger("s3proxy")
)

// Create a gin router
func NewGinEngine(ginMode string, version string, urlExpiration time.Duration, serverAPIKey string, s3Backend backend.Backend) *gin.Engine {

	gin.SetMode(ginMode)

	engine := gin.New()

	// Use middleware for logger, authorization
	engine.Use(middleware.NewLogger(log, "/"), middleware.NewRecovery(log), middleware.NewAuthorization(serverAPIKey, "/"))

	// health check
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"response": "ok", "version": version})
		return
	})

	presignedURLApiV1 := engine.Group("/api/v1/presigned/url")

	// create presigned url for a file upload
	presignedURLApiV1.POST("/:bucket/*key", func(c *gin.Context) {

		bucket := c.Param("bucket")
		key := c.Param("key")

		url, err := s3Backend.CreatePresignedURLForUpload(backend.BucketObject{BucketName: bucket, Key: key}, urlExpiration)
		if err != nil {
			log.Error("Failed to create presigned PutObject URL for %s %s", key, bucket, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PutObject URL for " + key})
			return
		}

		c.JSON(http.StatusOK, gin.H{"url": url})

		return
	})

	// create presigned url for a file download
	presignedURLApiV1.GET("/:bucket/*key", func(c *gin.Context) {

		var (
			bucket = c.Param("bucket")
			key    = c.Param("key")
		)

		url, err := s3Backend.CreatePresignedURLForDownload(backend.BucketObject{BucketName: bucket, Key: key}, urlExpiration)
		if err != nil {
			log.Error("Failed to create presigned GetObject URL for %s %s", key, bucket, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create GetObject URL for " + key})
			return
		}

		c.JSON(http.StatusOK, gin.H{"url": url})

		return
	})

	objectApiV1 := engine.Group("/api/v1/object")

	objectApiV1.DELETE("/:bucket/*key", func(c *gin.Context) {

		var (
			bucket = c.Param("bucket")
			key    = c.Param("key")
		)

		err := s3Backend.DeleteObject(backend.BucketObject{BucketName: bucket, Key: key})

		if err != nil {
			log.Error("Failed to delete object %s %s", key, bucket, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete object " + key})
			return
		}

		c.JSON(http.StatusOK, gin.H{"response": "ok"})

		return
	})

	objectApiV1.POST("/copy/:bucket/*key", func(c *gin.Context) {

		var (
			sourceBucket      = c.Param("bucket")
			sourceKey         = c.Param("key")
			destinationBucket = c.Query("destBucket")
			destinationKey    = c.Query("destKey")
		)

		if destinationBucket == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing destination bucket"})
			return
		}

		if destinationKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing destination key"})
			return
		}

		err := s3Backend.CopyObject(backend.BucketObject{BucketName: sourceBucket, Key: sourceKey},
			backend.BucketObject{BucketName: destinationBucket, Key: destinationKey})

		if err != nil {
			log.Error("Failed to copy object %s %s to %s %s %s", sourceBucket, sourceKey, destinationBucket, destinationKey, err)

			status, msg := http.StatusInternalServerError, fmt.Sprintf("Failed to copy object : sourceBucket=%q, sourceKey=%q", sourceBucket, sourceKey)

			if err, ok := err.(awserr.Error); ok {
				switch err.Code() {
				case s3.ErrCodeNoSuchBucket:
					status, msg = http.StatusNotFound, fmt.Sprintf("No such bucket : %q or %q", sourceBucket, destinationBucket)
				case s3.ErrCodeNoSuchKey:
					status, msg = http.StatusNotFound, fmt.Sprintf("No such key : %q", sourceKey)
				}
			}

			c.JSON(status, gin.H{"error": msg})

			return
		}

		c.JSON(http.StatusOK, gin.H{"response": "ok"})

		return
	})

	return engine
}
