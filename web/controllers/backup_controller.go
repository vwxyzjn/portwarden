package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"

	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden"
	"github.com/vwxyzjn/portwarden/web/models"
	"github.com/vwxyzjn/portwarden/web/utils"
)

func EncryptBackupHandler(c *gin.Context) {
	var ebi models.EncryptBackupInfo
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
	err = portwarden.CreateBackupFile(ebi.FileNamePrefix, ebi.Passphrase, sessionKey, models.BackupDefaultSleepMilliseconds)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": sessionKey})
		return
	}
}

//TODO: GoogleDriveHandler() will return Json with the google login url
// Not sure if it's supposed to call UploadFile() directly
func GoogleDriveHandler(c *gin.Context) {
	ctx := context.Background()
	credential, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	config, err := google.ConfigFromJSON(credential, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := utils.GetClient(ctx, config)
	token := utils.GetTokenFromWeb(config)
	// TODO: Assign encrypted data to fileBytes before uploadFile is called
	//UploadFile(fileBytes, client, token)

}

func DecryptBackupHandler(c *gin.Context) {
	var dbi models.DecryptBackupInfo
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
