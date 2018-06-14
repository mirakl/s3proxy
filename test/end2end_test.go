// +build end2end

// End to End test used for testing a version of a docker image of s3proxy
// with the following running backends : minio, rsyslog and s3proxy
package test

import (
	"testing"

	"github.com/mirakl/s3proxy/s3proxytest"
)

func TestEnd2End(t *testing.T) {
	defer s3proxytest.CatchPanic()

	s3proxyHost := "s3proxy:8080"

	s3proxytest.WaitForRessources(t, s3proxyHost)

	s3proxytest.RunSimpleScenarioForS3proxy(t, s3proxyHost)
}
