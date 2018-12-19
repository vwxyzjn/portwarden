package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/imdario/mergo"
	"github.com/vwxyzjn/portwarden/web"
	"golang.org/x/oauth2"
)

const (
	ErrBindingFromGin         = "(debugging message) error binding json"
	ErrRetrievingOauthCode    = "error retrieving oauth login credentials; try again"
	ErrCreatingPortwardenUser = "error creating a portwarden user"
	ErrGettingPortwardenUser  = "error getting a portwarden user"
	ErrMergingPortwardenUser  = "error merging a portwarden user"
	ErrLoginWithBitwarden     = "error logging in with Bitwarden"
	ErrSettingupBackup        = "error setting up backup"
	ErrBackupNotCancelled     = "error cancelling back up"

	MsgSuccessfullyCancelledBackingUp = "successfully cancelled backup process"

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
	opu.Email = pu.Email
	if err := opu.Get(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrGettingPortwardenUser})
		return
	}
	if err := mergo.Merge(&pu, opu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrMergingPortwardenUser})
		return
	}
	if err := pu.LoginWithBitwarden(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrLoginWithBitwarden})
		return
	}
	if err := pu.SetupAutomaticBackup(nil); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": MsgSuccessfullyCancelledBackingUp})
		return
	}
	if err := pu.Set(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrCreatingPortwardenUser})
		return
	}
}

func CancelEncryptBackupHandler(c *gin.Context) {
	var pu PortwardenUser
	var opu PortwardenUser
	if err := c.ShouldBindJSON(&pu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrBindingFromGin})
		return
	}
	if pu.BackupSetting.WillSetupBackup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "", "message": ErrBackupNotCancelled})
		return
	}
	opu.Email = pu.Email
	if err := opu.Get(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrGettingPortwardenUser})
		return
	}
	// Clean up users
	pu.BitwardenDataJSON = []byte{}
	pu.GoogleToken = &oauth2.Token{}
	pu.BitwardenSessionKey = ""
	pu.GoogleUserInfo = GoogleUserInfo{}
	pu.BackupSetting.WillSetupBackup = false
	if err := pu.Set(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrCreatingPortwardenUser})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": MsgSuccessfullyCancelledBackingUp})
}

//TODO: GoogleDriveHandler() will return Json with the google login url
// Not sure if it's supposed to call UploadFile() directly
func (ps *PortwardenServer) GetGoogleDriveLoginURLHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"login_url": web.GoogleDriveAppConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce),
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
	gui, err := RetrieveUserEmail(tok)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "message": "Login failure"})
		return
	}
	opu := PortwardenUser{Email: gui.Email}
	err = opu.Get()
	pu := PortwardenUser{GoogleToken: tok}
	if err := mergo.Merge(&pu, opu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrMergingPortwardenUser})
		return
	}
	if err != nil {
		if err != redis.Nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrGettingPortwardenUser})
			return
		}
		// Create a user
		err = pu.CreateWithGoogle()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrCreatingPortwardenUser})
			return
		}
		c.Redirect(http.StatusMovedPermanently, FrontEndBaseAddressTest+"home/"+"?access_token="+pu.GoogleToken.AccessToken+"&email="+pu.Email+"&will_setup_backup="+strconv.FormatBool(false))
		return
	}
	// Using info from exisiting user and update the access token
	if err := pu.Set(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": ErrCreatingPortwardenUser})
		return
	}
	c.Redirect(http.StatusMovedPermanently, FrontEndBaseAddressTest+"home/"+"?access_token="+pu.GoogleToken.AccessToken+"&email="+pu.Email+"&will_setup_backup="+strconv.FormatBool(pu.BackupSetting.WillSetupBackup))
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
