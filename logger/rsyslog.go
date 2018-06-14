package logger

import (
	"log/syslog"

	"github.com/op/go-logging"
)

// RSyslog backend for logging
type RSyslogBackend struct {
	Writer *syslog.Writer
}

// Create a new RSyslog backend
func NewRSyslogBackend(prefix string, host string, priority syslog.Priority, tag string) (b *RSyslogBackend, err error) {
	var w *syslog.Writer
	w, err = syslog.Dial("udp", host, priority, tag)
	return &RSyslogBackend{w}, err
}

// implements the Backend interface method
func (b *RSyslogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	line := rec.Formatted(calldepth + 1)
	switch level {
	case logging.CRITICAL:
		return b.Writer.Crit(line)
	case logging.ERROR:
		return b.Writer.Err(line)
	case logging.WARNING:
		return b.Writer.Warning(line)
	case logging.NOTICE:
		return b.Writer.Notice(line)
	case logging.INFO:
		return b.Writer.Info(line)
	case logging.DEBUG:
		return b.Writer.Debug(line)
	default:
	}
	panic("unhandled log level")
}
