package main

import (
	"fmt"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/vwxyzjn/portwarden"
	"github.com/vwxyzjn/portwarden/web"
	"github.com/vwxyzjn/portwarden/web/scheduler/server"
)

func main() {
	web.InitCommonVars()
	web.MachineryServer.RegisterTasks(map[string]interface{}{
		"BackupToGoogleDrive": BackupToGoogleDrive,
	})
	worker := web.MachineryServer.NewWorker("worker_name", 0)
	err := worker.Launch()
	if err != nil {
		panic(err)
	}
}

func BackupToGoogleDrive(email string) error {
	web.GlobalMutex.Lock()
	defer web.GlobalMutex.Unlock()
	name, err := os.Hostname()
	if err != nil {
		return err
	}
	fmt.Println("BackupToGoogleDrive called from worker:", name)
	pu := server.PortwardenUser{Email: email}
	err = pu.Get()
	if err != nil {
		return err
	}
	encryptedData, err := portwarden.CreateBackupBytesUsingBitwardenLocalJSON(pu.BitwardenDataJSON, web.BITWARDENCLI_APPDATA_DIR, pu.BackupSetting.Passphrase, pu.BitwardenSessionKey, web.BackupDefaultSleepMilliseconds)
	if err != nil {
		spew.Dump("BackupToGoogleDrive has an error", err)
		return err
	}
	newToken, err := server.UploadFile(encryptedData, pu.GoogleToken)
	if err != nil {
		return err
	}
	pu.GoogleToken = newToken
	eta := time.Now().UTC().Add(time.Second * time.Duration(pu.BackupSetting.BackupFrequencySeconds))
	err = pu.SetupAutomaticBackup(&eta)
	if err != nil {
		if err.Error() != server.ErrWillNotSetupBackupByUser {
			return err
		} else {
			fmt.Printf("user %v cancelled backup", pu.Email)
		}
	}
	// Update the access token
	err = pu.Set()
	if err != nil {
		return err
	}
	return nil
}
