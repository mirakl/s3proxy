package main

import (
	"os"
	"fmt"
	"time"
	"strings"
	"context"
	"net/http"
	"os/signal"
	"log/syslog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	rsyslog "s3proxy/logging"
	"s3proxy/middleware"
)

var (
	Version = "1.0.0"
	log                  = logging.MustGetLogger("s3proxy")
	format               = logging.MustStringFormatter(`%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x} %{message}`)
	presignedUrlduration = 15 * time.Minute
)

// Create presigned URL for uploading file to the bucket
func createPresignedPutObjectURL(bucket string, key string, svc *s3.S3) (string, error) {
	req, _ := svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return req.Presign(presignedUrlduration)
}

// Create presigned URL for downloading file from the bucket
func createPresignedGetObjectURL(bucket string, key string, svc *s3.S3) (string, error) {

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return req.Presign(presignedUrlduration)
}

// Delete object in a bucket
func deleteObject(bucket string, key string, svc *s3.S3) error {

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return err
}

// create s3 client with aws s3 or minio backend for integration tests running on local machine
func createS3Client(minio minioConfig) (*s3.S3, error) {

	var s3Config *aws.Config
	if minio.Host != "" {
		s3Config = &aws.Config{
			Credentials:      credentials.NewStaticCredentials(minio.AccessKey, minio.SecretKey, ""),
			Endpoint:         aws.String(minio.Host),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}
	}

	var sess *session.Session
	var err error

	// Initialize a session
	if s3Config != nil {
		sess, err = session.NewSession(s3Config)
	} else {
		sess, err = session.NewSession()
	}

	if err != nil {
		return nil, err
	}

	// Create S3 service client
	return s3.New(sess), nil
}

type minioConfig struct {
	Host      string
	AccessKey string
	SecretKey string
}

// GinEngine is gin router.
func GinEngine() *gin.Engine {

	minioConfig := minioConfig{
		Host:      viper.GetString("use-minio"),
		AccessKey: viper.GetString("minio-access-key"),
		SecretKey: viper.GetString("minio-secret-key"),
	}

	serverApiKey := viper.GetString("api-key")

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Use middleware for logger, authorization
	router.Use(middleware.Logger(log, "/"), middleware.Recovery(log), middleware.Authorization(serverApiKey, "/"))

	s3Client, err := createS3Client(minioConfig)
	if err != nil {
		log.Error("Cannot create s3 client %s", err)
		os.Exit(1)
	}

	// health check
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"response": "ok", "version": Version})
		return
	})

	presignedURLApiV1 := router.Group("/api/v1/presigned/url")

	// create presigned url for a file upload
	presignedURLApiV1.POST("/:bucket/*key", func(c *gin.Context) {

		bucket := c.Param("bucket")
		key := c.Param("key")

		url, err := createPresignedPutObjectURL(bucket, key, s3Client)
		if err != nil {
			log.Error("Failed to create presigned PutObject URL for %s %s", key, bucket, err)
			c.JSON(500, gin.H{"error": "Failed to create PutObject URL for " + key})
			return
		}

		c.JSON(200, gin.H{"url": url})

		return
	})

	// create presigned url for a file download
	presignedURLApiV1.GET("/:bucket/*key", func(c *gin.Context) {

		bucket := c.Param("bucket")
		key := c.Param("key")

		url, err := createPresignedGetObjectURL(bucket, key, s3Client)
		if err != nil {
			log.Error("Failed to create presigned GetObject URL for %s %s", key, bucket, err)
			c.JSON(500, gin.H{"error": "Failed to create GetObject URL for " + key})
			return
		}

		c.JSON(200, gin.H{"url": url})

		return
	})

	objectApiV1 := router.Group("/api/v1/object")

	objectApiV1.DELETE("/:bucket/*key", func(c *gin.Context) {

		bucket := c.Param("bucket")
		key := c.Param("key")

		err := deleteObject(bucket, key, s3Client)

		if err != nil {
			log.Error("Failed to delete object %s %s", key, bucket, err)
			c.JSON(500, gin.H{"error": "Failed to delete object " + key})
			return
		}

		c.JSON(200, gin.H{"response": "ok"})

		return
	})

	return router
}

// Initialize env and flag parameters
func initViper() {

	pflag.StringP("api-key", "x", "", "Define server side API key for API call authorization")
	viper.BindPFlag("api-key", pflag.Lookup("api-key"))
	viper.SetDefault("api-key", "")

	pflag.IntP("http-port", "p", 8080, "The port that the proxy binds to")
	viper.BindPFlag("http-port", pflag.Lookup("http-port"))
	viper.SetDefault("http-port", 8080)

	pflag.StringP("use-rsyslog", "r", "", "Add rsyslog as second logging destination by specifying the rsyslog host and port (ex. localhost:514)")
	viper.BindPFlag("use-rsyslog", pflag.Lookup("use-rsyslog"))
	viper.SetDefault("use-rsyslog", "")

	pflag.StringP("use-minio", "m", "", "Use minio as backend by specifying the minio server host and port (ex. localhost:9000)")
	viper.BindPFlag("use-minio", pflag.Lookup("use-minio"))
	viper.SetDefault("use-minio", "")

	pflag.StringP("minio-access-key", "a", "", "Minion AccessKey equivalent to a AWS_ACCESS_KEY_ID")
	viper.BindPFlag("minio-access-key", pflag.Lookup("minio-access-key"))
	viper.SetDefault("minio-access-key", "")

	pflag.StringP("minio-secret-key", "s", "", "Minion AccessKey equivalent to a AWS_SECRET_ACCESS_KEY")
	viper.BindPFlag("minio-secret-key", pflag.Lookup("minio-secret-key"))
	viper.SetDefault("minio-secret-key", "")

	pflag.Parse()

	viper.SetEnvPrefix("s3proxy")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
}

// Initialize logging to stdout with/without rsyslog
func initLogging() {

	useRsyslog := viper.GetString("use-rsyslog")

	// setup logging
	backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend1Formatter := logging.NewBackendFormatter(backend1, format)

	if useRsyslog != "" {
		backend2, _ := rsyslog.NewRSyslogBackendPriority("access_local_dev_central", useRsyslog, syslog.LOG_LOCAL0, "s3proxy")
		backend2Formatter := logging.NewBackendFormatter(backend2, format)
		logging.SetBackend(backend1Formatter, backend2Formatter)
	}
}

// Format for flag content manage secure sensitive content
func formatFlag(str string, secured bool) string {
	if str == "" {
		return "undefined"
	} else if secured {
		return str[0:len(str)/3] + "***..."
	}

	return str
}

// log info on startup
func logInfo() {
	log.Info("s3proxy version:%v port:%v rsyslog:%v minio:%v api-key:%v", Version,
		viper.GetInt("http-port"),
		formatFlag(viper.GetString("use-rsyslog"), false),
		formatFlag(viper.GetString("use-minio"), false),
		formatFlag(viper.GetString("api-key"), true),
	)
}

func main() {

	initViper()
	initLogging()

	addr := fmt.Sprintf(":%d", viper.GetInt("http-port")) // ":8080"

	router := GinEngine()

	logInfo()

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {

		log.Info("Listening ...")

		// service connections
		if err := srv.ListenAndServe(); err != nil {
			log.Error("Error: %s\n", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info("Shutdown Server ...")

	// wait max 5 seconds before killing
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown : ", err)
	}
	log.Info("Server exiting")
}
