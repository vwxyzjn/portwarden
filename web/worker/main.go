package main

import (
	"fmt"

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
	pu := server.PortwardenUser{Email: email}
	fmt.Println(pu.Email)
	err := pu.Get()
	if err != nil {
		return err
	}
	encryptedData, err := portwarden.CreateBackupBytesUsingBitwardenLocalJSON(pu.BitwardenDataJSON, web.BITWARDENCLI_APPDATA_DIR, pu.BackupSetting.Passphrase, pu.BitwardenSessionKey, web.BackupDefaultSleepMilliseconds)
	if err != nil {
		return err
	}
	err = server.UploadFile(encryptedData, pu.GoogleToken)
	if err != nil {
		return err
	}
	return nil
}
