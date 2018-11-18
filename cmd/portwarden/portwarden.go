package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"

	"github.com/vwxyzjn/portwarden"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	BackupFolderName              = "./portwarden_backup/"
	ErrVaultIsLocked              = "vault is locked"
	ErrNoPhassPhraseProvided      = "no passphrase provided"
	ErrNoFilenameProvided         = "no filename provided"
	ErrSessionKeyExtractionFailed = "session key extraction failed"

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
				err := EncryptBackupController(filename, passphrase)
				if err != nil {
					fmt.Println("encryption failed: " + err.Error())
					return err
				}
				fmt.Println("encrypted export successful")
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
				err := DecryptBackupController(filename, passphrase)
				if err != nil {
					fmt.Println("encryption failed: " + err.Error())
					return err
				}
				fmt.Println("decryption successful")
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

func EncryptBackupController(fileName, passphrase string) error {
	sessionKey, err := BWGetSessionKey()
	if err != nil {
		return err
	}
	return portwarden.CreateBackupFile(fileName, passphrase, sessionKey, sleepMilliseconds)
}

func DecryptBackupController(fileName, passphrase string) error {
	return portwarden.DecryptBackupFile(fileName, passphrase)
}

func BWGetSessionKey() (string, error) {
	sessionKey, err := BWUnlockVaultToGetSessionKey()
	if err != nil {
		if err.Error() == portwarden.BWErrNotLoggedIn {
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
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Println("An error occured: ", err)
	}
	cmd.Wait()
	sessionKey, err := portwarden.ExtractSessionKey(stdout.String())
	if err != nil {
		return "", errors.New(string(stdout.Bytes()))
	}
	return sessionKey, nil
}

func BWLoginGetSessionKey() (string, error) {
	cmd := exec.Command("bw", "login")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return "", err
	}
	cmd.Wait()
	sessionKey, err := portwarden.ExtractSessionKey(stdout.String())
	if err != nil {
		return "", errors.New(string(stdout.Bytes()))
	}
	return sessionKey, nil
}
