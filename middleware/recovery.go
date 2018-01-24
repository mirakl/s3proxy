package middleware

import (
	"net/http/httputil"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

func Recovery(log *logging.Logger) gin.HandlerFunc {
	return RecoveryWithLogger(log)
}

// Recovery returns a middleware that recovers from any panics and writes a 500 if there was one.
func RecoveryWithLogger(log *logging.Logger) gin.HandlerFunc {

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if log != nil {
					httprequest, _ := httputil.DumpRequest(c.Request, false)
					log.Info("[Recovery] panic recovered:%s %s", string(httprequest), err)
				}
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
