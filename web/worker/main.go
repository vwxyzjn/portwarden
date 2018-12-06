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
	time.Sleep(time.Millisecond * 400)
	web.GlobalMutex.Lock()
	defer web.GlobalMutex.Unlock()
	name, err := os.Hostname()
	if err != nil {
		return err
	}
	fmt.Println("BackupToGoogleDrive called from worker:", name)

	// Check whether user cancelled backup
	pu := server.PortwardenUser{Email: email}
	err = pu.Get()
	if err != nil {
		return err
	}
	if !pu.BackupSetting.WillSetupBackup {
		fmt.Printf("user %v cancelled backup \n", pu.Email)
		return nil
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

	// Check whether user cancelled backup
	opu := server.PortwardenUser{Email: email}
	err = opu.Get()
	if err != nil {
		return err
	}
	if !opu.BackupSetting.WillSetupBackup {
		fmt.Printf("user %v cancelled backup \n", pu.Email)
		return nil
	}

	eta := time.Now().UTC().Add(time.Second * time.Duration(pu.BackupSetting.BackupFrequencySeconds))
	err = pu.SetupAutomaticBackup(&eta)
	if err != nil {
		return err
	}
	// Update the access token
	err = pu.Set()
	if err != nil {
		return err
	}
	return nil
}
