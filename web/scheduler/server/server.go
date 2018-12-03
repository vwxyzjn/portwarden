package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type PortwardenServer struct {
	Port                      int
	Router                    *gin.Engine
	GoogleDriveContext        context.Context
	GoogleDriveAppCredentials []byte
	GoogleDriveAppConfig      *oauth2.Config
}

func (ps *PortwardenServer) Run() {
	ps.Router = gin.Default()
	ps.Router.Use(CORSMiddleware())

	ps.Router.GET("/", func(c *gin.Context) {
		c.JSON(200, "Welcome to Portwarden API")
	})

	ps.Router.POST("/decrypt", DecryptBackupHandler)
	ps.Router.GET("/gdrive/loginUrl", ps.GetGoogleDriveLoginURLHandler)

	ps.Router.GET("/gdrive/login", ps.GetGoogleDriveLoginHandler)

	ps.Router.Use(TokenAuthMiddleware())
	ps.Router.GET("/test/TokenAuthMiddleware", func(c *gin.Context) {
		c.JSON(200, "success")
	})
	ps.Router.POST("/encrypt", EncryptBackupHandler)
	ps.Router.POST("/encrypt/cancel", CancelEncryptBackupHandler)

	ps.Router.Run(":" + strconv.Itoa(ps.Port))
}
