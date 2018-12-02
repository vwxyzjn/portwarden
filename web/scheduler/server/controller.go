package server

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden/web"
	"golang.org/x/oauth2"
)

const (
	ErrBindingFromGin         = "(debugging message) error binding json"
	ErrRetrievingOauthCode    = "error retrieving oauth login credentials; try again"
	ErrCreatingPortwardenUser = "error creating a portwarden user"
	ErrGettingPortwardenUser  = "error getting a portwarden user"
	ErrLoginWithBitwarden     = "error logging in with Bitwarden"
	ErrSettingupBackup        = "error setting up backup"

	FrontEndBaseAddressTest = "http://localhost:8000/"
	FrontEndBaseAddressProd = ""
)

func EncryptBackupHandler(c *gin.Context) {
	var pu PortwardenUser
	var opu PortwardenUser
	if err := c.ShouldBindJSON(&pu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrBindingFromGin})
		return
	}
	if err := pu.LoginWithBitwarden(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrLoginWithBitwarden})
		return
	}
	opu.Email = pu.Email
	if err := opu.Get(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrGettingPortwardenUser})
		return
	}
	opu.BackupSetting = pu.BackupSetting
	if err := opu.SetupAutomaticBackup(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrSettingupBackup})
		return
	}
	if err := opu.Set(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrCreatingPortwardenUser})
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
	tok, err := web.GoogleDriveAppConfig.Exchange(oauth2.NoContext, gdc.Code)
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
