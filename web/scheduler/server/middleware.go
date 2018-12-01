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
		c.Next()
	}
}
