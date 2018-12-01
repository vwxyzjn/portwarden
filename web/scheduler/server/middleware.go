package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const (
	GoogleOauth2TokenContextVariableName = "GoogleOauth2TokenContextVariableName"
)

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("middleware called")
		code := c.Query("code")

		tok, err := GoogleDriveAppConfig.Exchange(oauth2.NoContext, code)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "message": "Login failure"})
			c.Abort()
			return
		}
		_, err = GetUserInfo(tok)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "message": "Login failure"})
			c.Abort()
			return
		}
		c.Set(GoogleOauth2TokenContextVariableName, tok)
		fmt.Println("middleware passed")
		c.Next()
	}
}
