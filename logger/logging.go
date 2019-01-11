package logger

import (
	"fmt"
	"log/syslog"
	"os"

	logging "github.com/op/go-logging"
)

var (
	logFormat              = logging.MustStringFormatter(`%{time:15:04:05.000} %{shortfunc} %{level:.4s} %{id:03x} %{message}`)
	backendStderrFormatter logging.Backend
)

// setup logging
func init() {
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendStderrFormatter = logging.NewBackendFormatter(backend, logFormat)

	logging.SetBackend(backendStderrFormatter)
}

// Add logging to stdout with rsyslog
func AddRsyslogBackend(host string) error {
	const (
		mtype     = "generic"
		mserver   = "s3proxy"
		menv      = "none"
		namespace = "central"
	)

	backend, err := NewRSyslogBackend(host, syslog.LOG_LOCAL0, fmt.Sprintf("%s_%s_%s_%s", mtype, mserver, menv, namespace))
	if err != nil {
		return err
	}

	backendRsyslogFormatter := logging.NewBackendFormatter(backend, logFormat)
	logging.SetBackend(backendStderrFormatter, backendRsyslogFormatter)

	return nil
}
