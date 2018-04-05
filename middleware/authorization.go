package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mirakl/s3proxy/util"
)

func respondWithError(code int, message string, c *gin.Context) {
	c.JSON(code, gin.H{"error": message})
	c.Abort()
}

// Creates authorization middleware for API auhtorization
func NewAuthorization(serverToken string, notsecured ...string) gin.HandlerFunc {

	skip := util.Array2map(notsecured...)

	return func(c *gin.Context) {

		path := c.Request.URL.Path

		// check if server token is defined and path is secured
		if _, shouldSkip := skip[path]; serverToken != "" && !shouldSkip {
			accessToken := c.Request.Header.Get("Authorization")

			if accessToken != serverToken {
				respondWithError(401, "Invalid API token", c)
				return
			}
		}

		c.Next()
	}
}
