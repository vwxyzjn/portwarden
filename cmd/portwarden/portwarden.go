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

	capturer "github.com/kami-zh/go-capturer"
	"github.com/mholt/archiver"
	"github.com/tidwall/pretty"
	"github.com/vwxyzjn/portwarden"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	BackupFolderName         = "./portwarden_backup/"
	ErrVaultIsLocked         = "vault is locked"
	ErrNoPhassPhraseProvided = "no passphrase provided"
	ErrNoFilenameProvided    = "no filename provided"

	BWErrNotLoggedIn = "You are not logged in."
	Salt             = `,(@0vd<)D6c3:5jI;4BZ(#Gx2IZ6B>`
)

func main() {
	var passphrase string
	var filename string
	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "passphrase",
			Usage:       "The passphrase that is used to encrypt/decrypt export of Bitwarden Vault",
			Destination: &passphrase,
		},
		cli.StringFlag{
			Name:        "filename",
			Usage:       "The name of the file you wish to export or decrypt",
			Destination: &filename,
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
	sessionKey := BWGetSessionKey()

	// save formmated json to "main.json"
	rawByte := BWListItemsRawBytes(sessionKey)
	if err := json.Unmarshal(rawByte, &pwes); err != nil {
		panic(err)
	}
	err := BWGetAllAttachments(BackupFolderName, sessionKey, pwes[:5])
	if err != nil {
		panic(err)
	}
	formattedByte := pretty.Pretty(rawByte)
	if err := ioutil.WriteFile(BackupFolderName+"main.json", formattedByte, 0644); err != nil {
		panic(err)
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	err = archiver.Zip.Write(writer, []string{BackupFolderName})
	if err != nil {
		panic(err)
	}

	// derive a key from the master password
	err = portwarden.EncryptFile(fileName, b.Bytes(), passphrase)
	if err != nil {
		panic(err)
	}

	// cleanup: delete temporary files
	err = os.RemoveAll(BackupFolderName)
	if err != nil {
		panic(err)
	}
}

func decryptBackup(fileName, passphrase string) {
	tb, err := portwarden.DecryptFile(fileName, passphrase)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(fileName+".decrypted"+".zip", tb, 0644); err != nil {
		panic(err)
	}
}

func extractSessionKey(stdout string) string {
	r := regexp.MustCompile(`BW_SESSION=".+"`)
	sessionKeyRawString := r.FindAllString(stdout, 1)[0]
	sessionKey := strings.TrimPrefix(sessionKeyRawString, `BW_SESSION="`)
	sessionKey = sessionKey[:len(sessionKey)-1]
	return sessionKey
}

func BWGetSessionKey() string {
	sessionKey, err := BWUnlockVaultToGetSessionKey()
	if err != nil {
		if err.Error() == BWErrNotLoggedIn {
			fmt.Println("try login")
			sessionKey, err = BWLoginGetSessionKey()
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
	return sessionKey
}

func BWUnlockVaultToGetSessionKey() (string, error) {
	var err error
	out := capturer.CaptureOutput(func() {
		cmd := exec.Command("bw", "unlock")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	})

	if err != nil {
		if string(out) == BWErrNotLoggedIn {
			return "", errors.New(string(out))
		} else {
			return "", err
		}
	}

	return extractSessionKey(string(out)), nil
}

func BWLoginGetSessionKey() (string, error) {
	var err error
	out := capturer.CaptureOutput(func() {
		cmd := exec.Command("bw", "login")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	})
	if err != nil {
		return "", err
	}
	return extractSessionKey(string(out)), nil
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
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
