package server

import (
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden/web/controllers"
)

type PortwardenServer struct {
	Port   int
	Router *gin.Engine
}

func (ps *PortwardenServer) Run() {
	ps.Router = gin.Default()
	ps.Router.Use(cors.Default())

	ps.Router.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	ps.Router.POST("/encrypt", controllers.EncryptBackupHandler)
	ps.Router.POST("/decrypt", controllers.DecryptBackupHandler)

	ps.Router.Run(":" + strconv.Itoa(ps.Port))
}
