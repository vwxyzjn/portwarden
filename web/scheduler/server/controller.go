package server

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const (
	ErrRetrievingOauthCode    = "error retrieving oauth login credentials; try again"
	ErrCreatingPortwardenUser = "error creating a portwarden user"
	ErrGettingPortwardenUser  = "error creating a portwarden user"
	ErrLoginWithBitwarden     = "error logging in with Bitwarden"

	FrontEndBaseAddressTest = "http://localhost:8000/"
	FrontEndBaseAddressProd = ""
)

func EncryptBackupHandler(c *gin.Context) {
	var pu PortwardenUser
	if err := c.ShouldBindJSON(&pu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrLoginWithBitwarden})
		return
	}
	if err := pu.LoginWithBitwarden(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrLoginWithBitwarden})
		return
	}
	pu.Get()
	fmt.Println(string(pu.BitwardenDataJSON))
	// sessionKey, err := portwarden.BWLoginGetSessionKey(&pu.BitwardenLoginCredentials)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": sessionKey})
	// 	return
	// }
	// err = portwarden.CreateBackupFile(pu.FileNamePrefix, pu.Passphrase, sessionKey, BackupDefaultSleepMilliseconds)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": sessionKey})
	// 	return
	// }
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
	pu := &PortwardenUser{GoogleToken: tok}
	err = pu.CreateWithGoogle()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrCreatingPortwardenUser})
		return
	}

	spew.Dump(pu)
	c.Redirect(http.StatusMovedPermanently, FrontEndBaseAddressTest+"?access_token="+pu.GoogleToken.AccessToken)
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
