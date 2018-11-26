package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	GoogleClient              *http.Client
}

func (ps *PortwardenServer) Run() {
	var err error
	ps.GoogleDriveContext = context.Background()
	ps.GoogleDriveAppConfig, err = google.ConfigFromJSON(ps.GoogleDriveAppCredentials, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	// ps.GoogleClient = GetClient(ps.GoogleDriveContext, ps.GoogleDriveAppConfig)

	ps.Router = gin.Default()
	ps.Router.Use(cors.Default())

	ps.Router.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	ps.Router.POST("/encrypt", EncryptBackupHandler)
	ps.Router.POST("/decrypt", DecryptBackupHandler)
	ps.Router.GET("/gdrive/loginUrl", ps.GetGoogleDriveLoginURLHandler)
	ps.Router.GET("/gdrive/login", ps.GetGoogleDriveLoginHandler)

	ps.Router.Run(":" + strconv.Itoa(ps.Port))
}
