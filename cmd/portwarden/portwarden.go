package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/tidwall/pretty"
	"github.com/vwxyzjn/portwarden"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	Salt                          = `,(@0vd<)D6c3:5jI;4BZ(#Gx2IZ6B>`
	BackupFolderName              = "./portwarden_backup/"
	ErrVaultIsLocked              = "vault is locked"
	ErrNoPhassPhraseProvided      = "no passphrase provided"
	ErrNoFilenameProvided         = "no filename provided"
	ErrSessionKeyExtractionFailed = "session key extraction failed"

	BWErrNotLoggedIn           = "You are not logged in."
	BWErrInvalidMasterPassword = "Invalid master password."
	BWEnterEmailAddress        = "? Email address:"
	BWEnterMasterPassword      = "? Master password:"
)

var (
	passphrase        string
	filename          string
	sleepMilliseconds int
)

func main() {
	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "passphrase",
			Usage:       "The passphrase that is used to encrypt/decrypt the backup export of your Bitwarden Vault",
			Destination: &passphrase,
		},
		cli.StringFlag{
			Name:        "filename",
			Usage:       "The name of the file you wish to export or decrypt",
			Destination: &filename,
		},
		cli.IntFlag{
			Name:        "sleep-milliseconds",
			Usage:       "The number of milliseconds before making another request to download attachment",
			Destination: &sleepMilliseconds,
			Value:       300,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "encrypt",
			Aliases: []string{"e"},
			Usage:   "Export the Bitwarden Vault with encryption to a `.portwarden` file",
			Action: func(c *cli.Context) error {
				if len(passphrase) == 0 {
					return errors.New(ErrNoPhassPhraseProvided)
				}
				encryptBackup(filename, passphrase)
				return nil
			},
		},
		{
			Name:    "decrypt",
			Aliases: []string{"d"},
			Usage:   "Decrypt a `.portwarden` file",
			Action: func(c *cli.Context) error {
				if len(passphrase) == 0 {
					return errors.New(ErrNoPhassPhraseProvided)
				}
				if len(filename) == 0 {
					return errors.New(ErrNoFilenameProvided)
				}
				decryptBackup(filename, passphrase)
				return nil
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func encryptBackup(fileName, passphrase string) {
	if !strings.HasSuffix(fileName, ".portwarden") {
		fileName += ".portwarden"
	}

	pwes := []portwarden.PortWardenElement{}
	sessionKey, err := BWGetSessionKey()
	if err != nil {
		fmt.Println("encryption failed: " + err.Error())
		return
	}

	// save formmated json to "main.json"
	rawByte := BWListItemsRawBytes(sessionKey)
	if err := json.Unmarshal(rawByte, &pwes); err != nil {
		fmt.Println("encryption failed: " + err.Error())
		return
	}
	err = BWGetAllAttachments(BackupFolderName, sessionKey, pwes[:5])
	if err != nil {
		fmt.Println("encryption failed: " + err.Error())
		return
	}
	formattedByte := pretty.Pretty(rawByte)
	if err := ioutil.WriteFile(BackupFolderName+"main.json", formattedByte, 0644); err != nil {
		fmt.Println("encryption failed: " + err.Error())
		return
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	err = archiver.Zip.Write(writer, []string{BackupFolderName})
	if err != nil {
		fmt.Println("encryption failed: " + err.Error())
		return
	}

	// derive a key from the master password
	err = portwarden.EncryptFile(fileName, b.Bytes(), passphrase)
	if err != nil {
		fmt.Println("encryption failed: " + err.Error())
		return
	}

	// cleanup: delete temporary files
	err = os.RemoveAll(BackupFolderName)
	if err != nil {
		fmt.Println("encryption failed: " + err.Error())
		return
	}
}

func decryptBackup(fileName, passphrase string) {
	tb, err := portwarden.DecryptFile(fileName, passphrase)
	if err != nil {
		fmt.Println("decryption failed: " + err.Error())
		return
	}
	if err := ioutil.WriteFile(fileName+".decrypted"+".zip", tb, 0644); err != nil {
		fmt.Println("decryption failed: " + err.Error())
		return
	}
}

func extractSessionKey(stdout string) (string, error) {
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

func BWGetSessionKey() (string, error) {
	sessionKey, err := BWUnlockVaultToGetSessionKey()
	if err != nil {
		if err.Error() == BWErrNotLoggedIn {
			sessionKey, err = BWLoginGetSessionKey()
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return sessionKey, err
}

func BWUnlockVaultToGetSessionKey() (string, error) {
	cmd := exec.Command("bw", "unlock")
	var stdout bytes.Buffer

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	defer stdin.Close()
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err = cmd.Start(); err != nil {
		fmt.Println("An error occured: ", err)
	}
	cmd.Wait()
	sessionKey, err := extractSessionKey(stdout.String())
	if err != nil {
		return "", errors.New(string(stdout.Bytes()))
	}
	return sessionKey, nil
}

func BWLoginGetSessionKey() (string, error) {
	cmd := exec.Command("bw", "login")
	var stdout bytes.Buffer

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	defer stdin.Close()
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err = cmd.Start(); err != nil {
		return "", err
	}
	cmd.Wait()
	sessionKey, err := extractSessionKey(stdout.String())
	if err != nil {
		return "", errors.New(string(stdout.Bytes()))
	}
	return sessionKey, nil
}

func BWListItemsRawBytes(sessionKey string) []byte {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("bw", "list", "items", "--session", sessionKey)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	return stdout.Bytes()
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

func BWGetAllAttachments(outputDir, sessionKey string, pws []portwarden.PortWardenElement) error {
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
