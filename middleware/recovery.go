package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"net/http"
	"net/http/httputil"
)

// Creates a Recovery middlware, in case of a fatal error returns 500
func NewRecovery(log *logging.Logger) gin.HandlerFunc {
	return newRecoveryWithLogger(log)
}

// Recovery returns a middleware that recovers from any panics and writes a 500 if there was one.
func newRecoveryWithLogger(log *logging.Logger) gin.HandlerFunc {

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if log != nil {
					httprequest, _ := httputil.DumpRequest(c.Request, false)
					log.Info("[Recovery] panic recovered:%s %s", string(httprequest), err)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
