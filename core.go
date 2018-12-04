package portwarden

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/tidwall/pretty"
)

const (
	BackupFolderName              = "./portwarden_backup/"
	ErrVaultIsLocked              = "vault is locked"
	ErrNoPhassPhraseProvided      = "no passphrase provided"
	ErrNoFilenameProvided         = "no filename provided"
	ErrSessionKeyExtractionFailed = "session key extraction failed"

	BWErrNotLoggedIn           = "You are not logged in."
	BWErrInvalidMasterPassword = "Invalid master password."
	BWEnterEmailAddress        = "? Email address:"
	BWEnterMasterPassword      = "? Master password:"

	LoginCredentialMethodNone          = 100
	LoginCredentialMethodAuthenticator = 0
	LoginCredentialMethodEmail         = 1
	LoginCredentialMethodYubikey       = 3
)

// LoginCredentials is used to login to the `bw` cli. See documentation
// https://help.bitwarden.com/article/cli/
// The possible `Method` values are
// None 			100
// Authenticator	0
// Email			1
// Yubikey			3
type LoginCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Method   int    `json:"method"`
	Code     string `json:"code"`
}

func CreateBackupBytesUsingBitwardenLocalJSON(dataJson []byte, BITWARDENCLI_APPDATA_DIR, passphrase, sessionKey string, sleepMilliseconds int) ([]byte, error) {
	// Put data.json in the BITWARDENCLI_APPDATA_DIR
	defer BWLogout()
	if err := ioutil.WriteFile(filepath.Join(BITWARDENCLI_APPDATA_DIR, "data.json"), dataJson, 0644); err != nil {
		return nil, err
	}
	return CreateBackupBytes(passphrase, sessionKey, sleepMilliseconds)
}

func CreateBackupFile(fileName, passphrase, sessionKey string, sleepMilliseconds int) error {
	defer BWLogout()
	if !strings.HasSuffix(fileName, ".portwarden") {
		fileName += ".portwarden"
	}
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	encryptedData, err := CreateBackupBytes(passphrase, sessionKey, sleepMilliseconds)
	if err != nil {
		return err
	}
	f.Write(encryptedData)
	return nil
}

func CreateBackupBytes(passphrase, sessionKey string, sleepMilliseconds int) ([]byte, error) {
	pwes := []PortWardenElement{}

	// save formmated json to "main.json"
	rawByte, err := BWListItemsRawBytes(sessionKey)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rawByte, &pwes); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(BackupFolderName, os.ModePerm); err != nil {
		return nil, err
	}
	err = BWGetAllAttachments(BackupFolderName, sessionKey, pwes, sleepMilliseconds)
	if err != nil {
		return nil, err
	}
	formattedByte := pretty.Pretty(rawByte)
	if err := ioutil.WriteFile(BackupFolderName+"main.json", formattedByte, 0644); err != nil {
		return nil, err
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	err = archiver.Zip.Write(writer, []string{BackupFolderName})
	if err != nil {
		return nil, err
	}

	// derive a key from the master password
	encryptedBytes, err := EncryptBytes(b.Bytes(), passphrase)
	if err != nil {
		return nil, err
	}

	// cleanup: delete temporary files
	err = os.RemoveAll(BackupFolderName)
	if err != nil {
		return nil, err
	}
	return encryptedBytes, nil
}

func DecryptBackupFile(fileName, passphrase string) error {
	rawBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	tb, err := DecryptBytes(rawBytes, passphrase)
	if err != nil {
		fmt.Println("decryption failed: " + err.Error())
		return err
	}
	if err := ioutil.WriteFile(fileName+".decrypted"+".zip", tb, 0644); err != nil {
		fmt.Println("decryption failed: " + err.Error())
		return err
	}
	return nil
}

func ExtractSessionKey(stdout string) (string, error) {
	r := regexp.MustCompile(`BW_SESSION=".+"`)
	matches := r.FindAllString(stdout, 1)
	if len(matches) == 0 {
		return "", errors.New(ErrSessionKeyExtractionFailed)
	}
	sessionKeyRawString := r.FindAllString(stdout, 1)[0]
	sessionKey := strings.TrimPrefix(sessionKeyRawString, `BW_SESSION="`)
	sessionKey = sessionKey[:len(sessionKey)-1]
	return sessionKey, nil
}

func BWListItemsRawBytes(sessionKey string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("bw", "list", "items", "--session", sessionKey)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func BWGetAttachment(outputDir, itemID, attachmentID, sessionKey string) error {
	cmd := exec.Command("bw", "get", "attachment", attachmentID, "--itemid", itemID,
		"--session", sessionKey, "--output", outputDir)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func BWGetAllAttachments(outputDir, sessionKey string, pws []PortWardenElement, sleepMilliseconds int) error {
	for _, item := range pws {
		if len(item.Attachments) > 0 {
			for _, innerItem := range item.Attachments {
				err := BWGetAttachment(outputDir+item.Name+"/", item.ID, innerItem.ID, sessionKey)
				time.Sleep(time.Millisecond * time.Duration(sleepMilliseconds))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func BWLoginGetSessionKey(lc *LoginCredentials) (string, error) {
	var cmd *exec.Cmd
	if lc.Method != LoginCredentialMethodNone {
		cmd = exec.Command("bw", "login", lc.Email, lc.Password, "--method", strconv.Itoa(lc.Method), "--code", lc.Code, "--raw")
	} else {
		cmd = exec.Command("bw", "login", lc.Email, lc.Password, "--raw")
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return stdout.String(), err
	}
	sessionKey := stdout.String()
	return sessionKey, nil
}

func BWLoginGetSessionKeyAndDataJSON(lc *LoginCredentials, BITWARDENCLI_APPDATA_DIR string) (string, []byte, error) {
	defer BWLogout()
	sessionKey, err := BWLoginGetSessionKey(lc)
	if err != nil {
		return "", nil, err
	}
	dataJSONPath := filepath.Join(BITWARDENCLI_APPDATA_DIR, "data.json")
	dat, err := ioutil.ReadFile(dataJSONPath)
	if err != nil {
		return "", nil, err
	}
	err = os.Remove(dataJSONPath)
	if err != nil {
		return "", nil, err
	}
	return sessionKey, dat, nil
}

func BWLogout() error {
	cmd := exec.Command("bw", "logout")
	return cmd.Run()
}
