package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mirakl/s3proxy/backend"
	"github.com/mirakl/s3proxy/logger"
	"github.com/mirakl/s3proxy/router"
	"github.com/op/go-logging"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "go.uber.org/automaxprocs"
)

var (
	version       = "undefined"
	log           = logging.MustGetLogger("s3proxy")
	urlExpiration = 15 * time.Minute
)

// Initialize env and flag parameters
func initViper() {

	pflag.StringP("api-key", "x", "", "Define server side API key for API call authorization")
	die(viper.BindPFlag("api-key", pflag.Lookup("api-key")))
	viper.SetDefault("api-key", "")

	pflag.IntP("http-port", "p", 8080, "The port that the proxy binds to")
	die(viper.BindPFlag("http-port", pflag.Lookup("http-port")))
	viper.SetDefault("http-port", 8080)

	pflag.StringP("use-rsyslog", "r", "", "Add rsyslog as second logging destination by specifying the rsyslog host and port (ex. localhost:514)")
	die(viper.BindPFlag("use-rsyslog", pflag.Lookup("use-rsyslog")))
	viper.SetDefault("use-rsyslog", "")

	pflag.StringP("use-minio", "m", "", "Use minio as backend by specifying the minio server host and port (ex. localhost:9000)")
	die(viper.BindPFlag("use-minio", pflag.Lookup("use-minio")))
	viper.SetDefault("use-minio", "")

	pflag.StringP("minio-access-key", "a", "", "Minion AccessKey equivalent to a AWS_ACCESS_KEY_ID")
	die(viper.BindPFlag("minio-access-key", pflag.Lookup("minio-access-key")))
	viper.SetDefault("minio-access-key", "")

	pflag.StringP("minio-secret-key", "s", "", "Minion AccessKey equivalent to a AWS_SECRET_ACCESS_KEY")
	die(viper.BindPFlag("minio-secret-key", pflag.Lookup("minio-secret-key")))
	viper.SetDefault("minio-secret-key", "")

	pflag.Parse()

	viper.SetEnvPrefix("s3proxy")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
}

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// log info on startup
func logStartupInfo() {
	formatFlag := func(str string, secured bool) string {
		if str == "" {
			return "undefined"
		} else if secured {
			return str[0:len(str)/3] + "***..."
		}

		return str
	}
	log.Infof("s3proxy version:%v port:%v rsyslog:%v minio:%v api-key:%v", version,
		viper.GetInt("http-port"),
		formatFlag(viper.GetString("use-rsyslog"), false),
		formatFlag(viper.GetString("use-minio"), false),
		formatFlag(viper.GetString("api-key"), true),
	)
}

func main() {
	initViper()

	useRsyslog := viper.GetString("use-rsyslog")
	if useRsyslog != "" {
		if err := logger.AddRsyslogBackend(useRsyslog); err != nil {
			log.Errorf("error %v", err)
		}
	}

	addr := fmt.Sprintf(":%d", viper.GetInt("http-port")) // ":8080"
	serverAPIKey := viper.GetString("api-key")

	var s3Backend backend.Backend
	var err error

	if viper.GetString("use-minio") != "" {
		minioBackendConfig := backend.S3BackendConfig{
			Host:             viper.GetString("use-minio"),
			AccessKey:        viper.GetString("minio-access-key"),
			SecretKey:        viper.GetString("minio-secret-key"),
			DisableSSL:       true, // For minio : True
			S3ForcePathStyle: true, // Form minio : True
		}

		s3Backend, err = backend.NewS3Backend(minioBackendConfig)
	} else {
		s3Backend, err = backend.NewS3Backend()
	}
	if err != nil {
		log.Errorf("Failed to intialize S3Backend : %v ", err)
		os.Exit(1)
	}

	router := router.NewGinEngine(gin.ReleaseMode, version, urlExpiration, serverAPIKey, s3Backend)

	router.RedirectTrailingSlash = false // return 404 when a <path> is not found instead redirecting to <path> + "/"

	logStartupInfo()

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Info("Listening ...")

		// service connections
		if err := srv.ListenAndServe(); err != nil {
			log.Errorf("Error: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info("Shutdown Server ...")

	// wait max 5 seconds before killing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown : %v", err)
	}

	log.Info("Server exiting")
}
