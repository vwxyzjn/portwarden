package server

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v2"
)

var (
	GoogleDriveAppConfig *oauth2.Config
	RedisClient          *redis.Client

	BITWARDENCLI_APPDATA_DIR string
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
	ps.GoogleDriveAppConfig, err = google.ConfigFromJSON(ps.GoogleDriveAppCredentials, "https://www.googleapis.com/auth/userinfo.profile", "email", drive.DriveScope)
	GoogleDriveAppConfig = ps.GoogleDriveAppConfig // quick hack
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	// ps.GoogleClient = GetClient(ps.GoogleDriveContext, ps.GoogleDriveAppConfig)

	// Setup Redis
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err = RedisClient.Ping().Result()
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Get Bitwarden CLI Env Var
	BITWARDENCLI_APPDATA_DIR = os.Getenv("BITWARDENCLI_APPDATA_DIR")

	ps.Router = gin.Default()
	ps.Router.Use(cors.Default())

	ps.Router.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	ps.Router.POST("/decrypt", DecryptBackupHandler)
	ps.Router.GET("/gdrive/loginUrl", ps.GetGoogleDriveLoginURLHandler)

	ps.Router.GET("/gdrive/login", ps.GetGoogleDriveLoginHandler)

	ps.Router.Use(TokenAuthMiddleware())
	ps.Router.GET("/test/TokenAuthMiddleware", func(c *gin.Context) {
		c.JSON(200, "success")
	})
	ps.Router.POST("/encrypt", EncryptBackupHandler)

	ps.Router.Run(":" + strconv.Itoa(ps.Port))
}
