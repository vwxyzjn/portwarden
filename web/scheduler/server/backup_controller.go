package server

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden"
	"golang.org/x/oauth2"
)

const (
	ErrRetrievingOauthCode = "error retrieving oauth login credentials; try again"
)

func EncryptBackupHandler(c *gin.Context) {
	var ebi EncryptBackupInfo
	if err := c.ShouldBindJSON(&ebi); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ""})
		return
	}
	sessionKey, err := portwarden.BWLoginGetSessionKey(&ebi.BitwardenLoginCredentials)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": sessionKey})
		return
	}
	err = portwarden.CreateBackupFile(ebi.FileNamePrefix, ebi.Passphrase, sessionKey, BackupDefaultSleepMilliseconds)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": sessionKey})
		return
	}
}

//TODO: GoogleDriveHandler() will return Json with the google login url
// Not sure if it's supposed to call UploadFile() directly
func (ps *PortwardenServer) GetGoogleDriveLoginURLHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"login_url": ps.GoogleDriveAppConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce),
	})
	return
}

func (ps *PortwardenServer) GetGoogleDriveLoginHandler(c *gin.Context) {
	var gdc GoogleDriveCredentials
	if err := c.ShouldBind(&gdc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrRetrievingOauthCode})
		return
	}
	tok, err := GoogleDriveAppConfig.Exchange(oauth2.NoContext, gdc.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "message": "Login failure"})
		return
	}
	// pu := &PortwardenUser{GoogleToken: tok}
	body, err := GetUserInfo(tok)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ""})
		return
	}
	spew.Dump(string(body))
	c.JSON(200, "Login Successful")
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
	}
}
