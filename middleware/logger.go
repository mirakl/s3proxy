package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"s3proxy/util"
	"time"
)

// Create a Logger middleware for gin
func NewLogger(log *logging.Logger, notlogged ...string) gin.HandlerFunc {
	return newLoggerWithWriter(log, notlogged...)
}

// Logger middleware that will write the access logs
func newLoggerWithWriter(log *logging.Logger, notlogged ...string) gin.HandlerFunc {

	skip := util.Array2map(notlogged...)

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Log only when path is not being skipped
		if _, shouldSkip := skip[path]; !shouldSkip {
			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			clientIP := c.ClientIP()
			method := c.Request.Method
			statusCode := c.Writer.Status()
			comment := c.Errors.ByType(gin.ErrorTypeAny).String()

			log.Info("%v | %3d | %13v | %15s | %s %s\n%s",
				end.Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				method,
				path,
				comment,
			)
		}
	}
}
