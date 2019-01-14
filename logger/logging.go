package logger

import (
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
	backend, err := NewRSyslogBackend("s3proxy", host, syslog.LOG_LOCAL0, "s3proxy_none_none_central")
	if err != nil {
		return err
	}

	backendRsyslogFormatter := logging.NewBackendFormatter(backend, logFormat)
	logging.SetBackend(backendStderrFormatter, backendRsyslogFormatter)

	return nil
}
