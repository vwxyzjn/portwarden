package server

import (
	"mime/multipart"

	"github.com/vwxyzjn/portwarden"
)

const (
	BackupDefaultSleepMilliseconds = 300
)

type EncryptBackupInfo struct {
	FileNamePrefix            string                      `json:"filename_prefix"`
	Passphrase                string                      `json:"passphrase"`
	BitwardenLoginCredentials portwarden.LoginCredentials `json:"bitwarden_login_credentials"`
}

type DecryptBackupInfo struct {
	File       *multipart.FileHeader `form:"file"`
	Passphrase string                `form:"passphrase"`
}
