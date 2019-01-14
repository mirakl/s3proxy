// +build integration

// Integration test used for testing s3proxy code of this repository
// with the following running backends : minio and rsyslog
package test

import (
	"net"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mirakl/s3proxy/backend"
	"github.com/mirakl/s3proxy/logger"
	"github.com/mirakl/s3proxy/router"
	"github.com/mirakl/s3proxy/s3proxytest"
	logging "github.com/op/go-logging"
)

var (
	log         = logging.MustGetLogger("s3proxy")
	s3proxyHost = ""
)

func setupIntegration(t *testing.T) {

	// Let the os select an available port for the s3proxy
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	s3proxyHost = listener.Addr().String()

	err = logger.AddRsyslogBackend("rsyslog:514")
	if err != nil {
		log.Fatal(err)
	}

	s3Backend, err := backend.NewS3Backend(s3proxytest.MinioBackendConfig)
	if err != nil {
		log.Fatal(err)
	}

	router := router.NewGinEngine(gin.ReleaseMode, "9.9.9", s3proxytest.UrlExpiration, s3proxytest.ServerAPIKey, s3Backend)

	go func() {
		log.Debug("Listening on port : %v", listener.Addr())

		// serve connections
		if err := http.Serve(listener, router); err != nil {
			log.Fatal(err)
		}
	}()

	s3proxytest.WaitForRessources(t, s3proxyHost)
}

func TestIntegration(t *testing.T) {
	defer s3proxytest.CatchPanic()

	setupIntegration(t)

	s3proxytest.RunSimpleScenarioForS3proxy(t, s3proxyHost)
}
