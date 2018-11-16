package models

import (
	"github.com/vwxyzjn/portwarden"
)

const (
	BackupDefaultSleepMilliseconds = 300
)

type BackupInfo struct {
	FileNamePrefix            string                      `json:"filename_prefix"`
	Passphrase                string                      `json:"passphrase"`
	BitwardenLoginCredentials portwarden.LoginCredentials `json:"bitwarden_login_credentials"`
}
