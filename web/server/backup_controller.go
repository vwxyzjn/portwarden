package server

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/oauth2"

	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden"
)

func EncryptBackupHandler(c *gin.Context) {
	var ebi EncryptBackupInfo
	if err := c.ShouldBindJSON(&ebi); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ""})
		return
	}

	spew.Dump(&ebi.BitwardenLoginCredentials)
	sessionKey, err := portwarden.BWLoginGetSessionKey(&ebi.BitwardenLoginCredentials)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": sessionKey})
		return
	}
	fmt.Println(sessionKey)
	err = portwarden.CreateBackupFile(ebi.FileNamePrefix, ebi.Passphrase, sessionKey, BackupDefaultSleepMilliseconds)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": sessionKey})
		return
	}
}

//TODO: GoogleDriveHandler() will return Json with the google login url
// Not sure if it's supposed to call UploadFile() directly
func (ps *PortwardenServer) GoogleDriveLoginHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"login_url": ps.GoogleDriveAppConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline),
	})
	return
}

func DecryptBackupHandler(c *gin.Context) {
	var dbi DecryptBackupInfo
	var err error
	if err = c.ShouldBind(&dbi); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ""})
		return
	}
	if dbi.File, err = c.FormFile("file"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ""})
		spew.Dump(gin.H{"error": err.Error(), "message": ""})
	}
}
