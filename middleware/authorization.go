package middleware

import (
	"github.com/gin-gonic/gin"
)

func respondWithError(code int, message string, c *gin.Context) {
	c.JSON(code, gin.H{"error": message})
	c.Abort()
}

// Manage api authorization key based on header token 'Authorization'
func Authorization(serverToken string, notsecured ...string) gin.HandlerFunc {

	skip := array2map(notsecured...)

	return func(c *gin.Context) {

		path := c.Request.URL.Path

		// check if server token is defined and path is secured
		if _, shouldSkip := skip[path]; serverToken != "" && !shouldSkip {
			accessToken := c.Request.Header.Get("Authorization")

			if accessToken == "" {
				respondWithError(401, "API token required", c)
				return
			}

			if accessToken != serverToken {
				respondWithError(401, "Invalid API token", c)
				return
			}
		}

		c.Next()
	}
}
