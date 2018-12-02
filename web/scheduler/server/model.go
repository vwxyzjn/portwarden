package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/vwxyzjn/portwarden"
	"github.com/vwxyzjn/portwarden/web"
	"golang.org/x/oauth2"
)

type BackupSetting struct {
	Passphrase             string `json:"passphrase"`
	BackupFrequencySeconds int    `json:"backup_frequency_seconds"`
}

type DecryptBackupInfo struct {
	File       *multipart.FileHeader `form:"file"`
	Passphrase string                `form:"passphrase"`
}

type GoogleTokenVerifyResponse struct {
	IssuedTo      string `json:"issued_to"`
	Audience      string `json:"audience"`
	UserID        string `json:"user_id"`
	Scope         string `json:"scope"`
	ExpiresIn     int64  `json:"expires_in"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	AccessType    string `json:"access_type"`
}

type GoogleDriveCredentials struct {
	State string `form:"state"`
	Code  string `form:"code"`
	Scope string `form:"scope"`
}

type PortwardenUser struct {
	Email                     string                       `json:"email"`
	BitwardenDataJSON         []byte                       `json:"bitwarden_data_json"`
	BitwardenSessionKey       string                       `json:"bitwarden_session_key"`
	BackupSetting             BackupSetting                `json:"backup_setting"`
	BitwardenLoginCredentials *portwarden.LoginCredentials `json:"bitwarden_login_credentials"` // Not stored in Redis
	GoogleUserInfo            GoogleUserInfo
	GoogleToken               *oauth2.Token
}

type GoogleUserInfo struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
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
	GoogleDriveClient := web.GoogleDriveAppConfig.Client(oauth2.NoContext, pu.GoogleToken)
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
	pu.Email = pu.GoogleUserInfo.Email
	err = pu.Set()
	if err != nil {
		return err
	}
	return nil
}

func (pu *PortwardenUser) LoginWithBitwarden() error {
	web.GlobalMutex.Lock()
	defer web.GlobalMutex.Unlock()
	opu := PortwardenUser{Email: pu.Email}
	err := opu.Get()
	if err != nil {
		return err
	}
	opu.BitwardenSessionKey, opu.BitwardenDataJSON, err = portwarden.BWLoginGetSessionKeyAndDataJSON(pu.BitwardenLoginCredentials, web.BITWARDENCLI_APPDATA_DIR)
	if err != nil {
		return err
	}
	err = opu.Set()
	if err != nil {
		return err
	}
	return nil
}

func (pu *PortwardenUser) SetupAutomaticBackup(eta *time.Time) error {
	signature := &tasks.Signature{
		Name: "BackupToGoogleDrive",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: pu.Email,
			},
		},
		ETA:        eta,
		RetryCount: web.MachineryRetryCount,
	}
	_, err := web.MachineryServer.SendTask(signature)
	if err != nil {
		return err
	}
	return nil
}

func (pu *PortwardenUser) Set() error {
	pu.BitwardenLoginCredentials = &portwarden.LoginCredentials{}
	puJson, err := json.Marshal(pu)
	if err != nil {
		return err
	}
	err = web.RedisClient.Set(pu.Email, string(puJson), 0).Err()
	if err != nil {
		panic(err)
	}
	return nil
}

func (pu *PortwardenUser) Get() error {
	val, err := web.RedisClient.Get(pu.Email).Result()
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(val), &pu); err != nil {
		return err
	}
	return nil
}

func VerifyGoogleAccessToekn(access_token string) (bool, error) {
	url := "https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=" + access_token
	response, err := http.Get(url)
	defer response.Body.Close()
	if err != nil {
		return false, err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}
	var gtvr GoogleTokenVerifyResponse
	if err := json.Unmarshal(body, &gtvr); err != nil {
		return false, err
	}
	if !gtvr.VerifiedEmail {
		return false, errors.New(string(body))
	}
	return true, nil
}
