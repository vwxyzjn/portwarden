package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	GoogleOauth2TokenContextVariableName = "GoogleOauth2TokenContextVariableName"
)

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		HeaderAuthorization, ok := c.Request.Header["Authorization"]
		if ok && len(HeaderAuthorization) >= 1 {
			token = HeaderAuthorization[0]
			token = strings.TrimPrefix(token, "Bearer ")
		}

		verified, err := VerifyGoogleAccessToekn(token)
		if err != nil || !verified {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "message": "Verification status is " + strconv.FormatBool(verified)})
			c.Abort()
			return
		}
		c.Set(GoogleOauth2TokenContextVariableName, token)
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
		} else {
			c.Next()
		}
	}
}
