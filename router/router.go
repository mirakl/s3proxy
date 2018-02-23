package router

import (
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"net/http"
	"s3proxy/backend"
	"s3proxy/middleware"
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

		url, err := s3Backend.CreatePresignedURLForUpload(bucket, key, urlExpiration)
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

		bucket := c.Param("bucket")
		key := c.Param("key")

		url, err := s3Backend.CreatePresignedURLForDownload(bucket, key, urlExpiration)
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

		bucket := c.Param("bucket")
		key := c.Param("key")

		err := s3Backend.DeleteObject(bucket, key)

		if err != nil {
			log.Error("Failed to delete object %s %s", key, bucket, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete object " + key})
			return
		}

		c.JSON(http.StatusOK, gin.H{"response": "ok"})

		return
	})

	return engine
}
