package server

import (
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/vwxyzjn/portwarden"
	"golang.org/x/oauth2"
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

type GoogleDriveCredentials struct {
	State string `form:"state"`
	Code  string `form:"code"`
	Scope string `form:"scope"`
}

type PortwardenUser struct {
	ID                  string
	GoogleUserInfo      GoogleUserInfo
	GoogleToken         *oauth2.Token
	BitwardenDataJSON   []byte
	BitwardenSessionKey string
}

type GoogleUserInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Link       string `json:"link"`
	Picture    string `json:"picture"`
	Locale     string `json:"locale"`
}

func (pu *PortwardenUser) CreateWithGoogle() error {
	postURL := "https://www.googleapis.com/oauth2/v2/userinfo"
	request, err := http.NewRequest("GET", postURL, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Host", "www.googleapis.com")
	request.Header.Add("Authorization", "Bearer "+pu.GoogleToken.AccessToken)
	request.Header.Add("Content-Length", strconv.FormatInt(request.ContentLength, 10))

	// For debugging
	//fmt.Println(request)
	GoogleDriveClient := GoogleDriveAppConfig.Client(oauth2.NoContext, pu.GoogleToken)
	response, err := GoogleDriveClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &pu.GoogleUserInfo); err != nil {
		return err
	}
	pu.ID = pu.GoogleUserInfo.ID
	return nil
}
