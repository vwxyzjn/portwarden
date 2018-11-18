package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden/web/controllers"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v2"
)

type PortwardenServer struct {
	Port                      int
	Router                    *gin.Engine
	GoogleDriveContext        context.Context
	GoogleDriveAppCredentials []byte
	GoogleDriveAppConfig      *oauth2.Config
}

func (ps *PortwardenServer) Run() {
	var err error
	ps.GoogleDriveContext = context.Background()
	ps.GoogleDriveAppConfig, err = google.ConfigFromJSON(ps.GoogleDriveAppCredentials, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	ps.Router = gin.Default()
	ps.Router.Use(cors.Default())

	ps.Router.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	ps.Router.POST("/encrypt", controllers.EncryptBackupHandler)
	ps.Router.POST("/decrypt", controllers.DecryptBackupHandler)

	ps.Router.Run(":" + strconv.Itoa(ps.Port))
}
