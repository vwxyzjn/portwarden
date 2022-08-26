package portwarden

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	b64 "encoding/base64"
	"github.com/davecgh/go-spew/spew"

	"github.com/mholt/archiver"
	"github.com/tidwall/pretty"
)

const (
	BackupFolderName              = "./portwarden_backup/"
	ErrVaultIsLocked              = "vault is locked"
	ErrNoPhassPhraseProvided      = "no passphrase provided"
	ErrNoFilenameProvided         = "no filename provided"
	ErrSessionKeyExtractionFailed = "session key extraction failed"
	ErrVaultNotEmptyForRestore    = "account's valut not empty! you have to restore the backup to an empty Bitwarden account"

	BWErrNotLoggedIn           = "You are not logged in."
	BWErrInvalidMasterPassword = "Invalid master password."
	BWEnterEmailAddress        = "? Email address:"
	BWEnterMasterPassword      = "? Master password:"

	LoginCredentialMethodNone          = 100
	LoginCredentialMethodAuthenticator = 0
	LoginCredentialMethodEmail         = 1
	LoginCredentialMethodYubikey       = 3

	ItemsJsonFileName   = "items.json"
	FoldersJSONFileName = "folders.json"
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
	defer BWDelete(BITWARDENCLI_APPDATA_DIR)
	if err := ioutil.WriteFile(filepath.Join(BITWARDENCLI_APPDATA_DIR, "data.json"), dataJson, 0644); err != nil {
		return nil, err
	}
	return CreateBackupBytes(passphrase, sessionKey, sleepMilliseconds)
}

func CreateBackupFile(fileName, passphrase, sessionKey string, sleepMilliseconds int, noLogout bool) error {
	if !noLogout {
		defer BWLogout()
	}
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
	if err := os.MkdirAll(BackupFolderName, os.ModePerm); err != nil {
		return nil, err
	}
	defer os.RemoveAll(BackupFolderName)

	// save formmated json to FoldersJSONFileName
	rawByte, err := BWListFoldersRawBytes(sessionKey)
	if err != nil {
		return nil, err
	}
	formattedByte := pretty.Pretty(rawByte)
	if err := ioutil.WriteFile(BackupFolderName+FoldersJSONFileName, formattedByte, 0644); err != nil {
		return nil, err
	}

	// save formmated json to ItemsJsonFileName
	rawByte, err = BWListItemsRawBytes(sessionKey)
	if err != nil {
		return nil, err
	}
	formattedByte = pretty.Pretty(rawByte)
	if err := ioutil.WriteFile(BackupFolderName+ItemsJsonFileName, formattedByte, 0644); err != nil {
		return nil, err
	}

	// download attachments
	pwes := []PortWardenElement{}
	if err := json.Unmarshal(rawByte, &pwes); err != nil {
		return nil, err
	}
	err = BWGetAllAttachments(BackupFolderName, sessionKey, pwes, sleepMilliseconds)
	if err != nil {
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

func RestoreBackupFile(fileName, passphrase, sessionKey string, sleepMilliseconds int, noLogout bool) error {
	// dummy check if the account is not empty, don't restore
	var err error
	var rawByte []byte

	var file []byte
	err = DecryptBackupFile(fileName, passphrase)
	if err != nil {
		return err
	}
	err = Unzip(fileName+".decrypted"+".zip", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(BackupFolderName)
	defer os.Remove(fileName + ".decrypted" + ".zip")

	rawByte, err = BWListItemsRawBytes(sessionKey)
	if err != nil {
		return err
	}
	pwes := []PortWardenElement{}
	if err := json.Unmarshal(rawByte, &pwes); err != nil {
		return err
	}
	if len(pwes) != 0 {
		return errors.New(ErrVaultNotEmptyForRestore)
	}

	// restore folders
	if file, err = ioutil.ReadFile(BackupFolderName + FoldersJSONFileName); err != nil {
		return err
	}
	folderData := PortWardenFolder{}
	err = json.Unmarshal([]byte(file), &folderData)
	if err != nil {
		return err
	}
	oldToNewFolderID := make(map[string]string)
	var itemBytes []byte
	for _, item := range folderData {
		time.Sleep(time.Millisecond * time.Duration(sleepMilliseconds))
		if item.ID != nil {
			itemBytes, err = json.Marshal(item)
			cmd := exec.Command("bw", "create", "folder", "--session", sessionKey, b64.StdEncoding.EncodeToString(itemBytes))
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err != nil {
				fmt.Println("An error occured: ", err)
				spew.Dump(stdout, stderr)
			}
			fmt.Println("restoring folder", item.Name)
			newItem := PortWardenFolderElement{}
			err = json.Unmarshal(stdout.Bytes(), &newItem)
			if err != nil {
				return err
			}
			oldToNewFolderID[*item.ID] = *newItem.ID
		}
	}

	// restore items
	if file, err = ioutil.ReadFile(BackupFolderName + ItemsJsonFileName); err != nil {
		return err
	}
	itemData := PortWarden{}
	err = json.Unmarshal([]byte(file), &itemData)
	if err != nil {
		return err
	}
	oldToNewItemID := make(map[string]string)
	for _, item := range itemData {
		time.Sleep(time.Millisecond * time.Duration(sleepMilliseconds))
		// deal with attachments separately
		item.Attachments = nil
		if item.FolderID != nil {
			*item.FolderID = oldToNewFolderID[*item.FolderID]
		}
		if item.OrganizationID == nil {
			// probably not needed
			item.CollectionIDS = nil

			itemBytes, err = json.Marshal(item)
			cmd := exec.Command("bw", "create", "item", "--session", sessionKey, b64.StdEncoding.EncodeToString(itemBytes))
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err != nil {
				fmt.Println("An error occured: ", err)
				spew.Dump(stdout, stderr)
			}
			fmt.Println("restoring item", item.Name)
			newItem := PortWardenElement{}
			err = json.Unmarshal(stdout.Bytes(), &newItem)
			if err != nil {
				return err
			}
			oldToNewItemID[item.ID] = newItem.ID
		}
	}
	fmt.Println("restoring item finished")

	// restore item's attachments
	if file, err = ioutil.ReadFile(BackupFolderName + ItemsJsonFileName); err != nil {
		return err
	}
	itemData = PortWarden{}
	err = json.Unmarshal([]byte(file), &itemData)
	if err != nil {
		return err
	}
	for _, item := range itemData {
		if len(item.Attachments) > 0 {
			time.Sleep(time.Millisecond * time.Duration(sleepMilliseconds))
			for _, innerItem := range item.Attachments {
				itemBytes, err = json.Marshal(item)
				cmd := exec.Command("bw", "create", "attachment", "--itemid", oldToNewItemID[item.ID], "--session", sessionKey, "--file", BackupFolderName+item.Name+"/"+innerItem.FileName)
				var stdout bytes.Buffer
				var stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				cmd.Stdin = os.Stdin
				if err := cmd.Run(); err != nil {
					fmt.Println("An error occured: ", err)
					spew.Dump(stdout, stderr)
				}
				fmt.Println("restoring item's attachment", item.Name, innerItem.FileName)
			}
		}
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

func BWListFoldersRawBytes(sessionKey string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("bw", "list", "folders", "--session", sessionKey)
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
				ourputDir := strings.TrimSpace(outputDir + item.Name) // Keep this line. See https://github.com/vwxyzjn/portwarden/issues/10
				err := BWGetAttachment(ourputDir+"/", item.ID, innerItem.ID, sessionKey)
				time.Sleep(time.Millisecond * time.Duration(sleepMilliseconds))
				if err != nil {
					spew.Dump(err, "failed item ids are ", item.ID, innerItem.ID, item.Name)
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
	sessionKey, err := BWLoginGetSessionKey(lc)
	if err != nil {
		return "", nil, err
	}
	defer BWDelete(BITWARDENCLI_APPDATA_DIR)
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
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("bw", "logout")
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(string(stderr.Bytes()))
	}
	return nil
}

func BWDelete(BITWARDENCLI_APPDATA_DIR string) error {
	dataJSONPath := filepath.Join(BITWARDENCLI_APPDATA_DIR, "data.json")
	err := os.Remove(dataJSONPath)
	if err != nil {
		return err
	}
	return nil
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
