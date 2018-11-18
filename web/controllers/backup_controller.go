package controllers

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"

	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden"
	"github.com/vwxyzjn/portwarden/web/models"
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
