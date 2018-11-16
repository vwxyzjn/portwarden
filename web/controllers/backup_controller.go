package controllers

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"

	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden"
	"github.com/vwxyzjn/portwarden/web/models"
)

func EncryptBackupController(c *gin.Context) {
	var bi models.BackupInfo
	if err := c.ShouldBindJSON(&bi); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	spew.Dump(&bi.BitwardenLoginCredentials)
	sessionKey, err := portwarden.BWLoginGetSessionKey(&bi.BitwardenLoginCredentials)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(sessionKey)
	err = portwarden.CreateBackupFile(bi.FileNamePrefix, bi.Passphrase, sessionKey, models.BackupDefaultSleepMilliseconds)
	if err != nil {
		panic(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}
