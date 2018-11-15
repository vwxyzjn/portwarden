package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden/web/controllers"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	r.GET("/ws", controllers.EncryptBackupController)

	r.Run(":5000")
}
