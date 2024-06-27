package main

import (
	"bytes"
	"encoding/json"
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
	ErrSessionKeyGetEnvFailed     = "no BW_SESSION found in environment variable"

	BWErrInvalidMasterPassword = "Invalid master password."
	BWEnterEmailAddress        = "? Email address:"
	BWEnterMasterPassword      = "? Master password:"
)

var (
	passphrase        string
	filename          string
	sleepMilliseconds int
	noLogout          bool
)

func main() {
	app := cli.NewApp()

	app.Version = "1.0.0"

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
		cli.BoolFlag{
			Name:        "no-logout",
			Usage:       "If set to true, then Portwarden won't log you out of the Bitwarden CLI",
			Destination: &noLogout,
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
					return err
				}
				fmt.Println("decryption successful")
				return nil
			},
		},
		{
			Name:    "restore",
			Aliases: []string{"d"},
			Usage:   "restore a `.portwarden` backgup to a Bitwarden Account",
			Action: func(c *cli.Context) error {
				if len(passphrase) == 0 {
					return errors.New(ErrNoPhassPhraseProvided)
				}
				if len(filename) == 0 {
					return errors.New(ErrNoFilenameProvided)
				}
				err := RestoreBackupController(filename, passphrase)
				if err != nil {
					return err
				}
				fmt.Println("restore successful")
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
	return portwarden.CreateBackupFile(fileName, passphrase, sessionKey, sleepMilliseconds, noLogout)
}

func DecryptBackupController(fileName, passphrase string) error {
	return portwarden.DecryptBackupFile(fileName, passphrase)
}

func RestoreBackupController(fileName, passphrase string) error {
	var err error
	var sessionKey string
	err = portwarden.BWLogout()
	if err != nil {
		if err.Error() != portwarden.BWErrNotLoggedIn {
			return err
		}
	}
	sessionKey, err = BWGetSessionKey()
	if err != nil {
		return err
	}
	return portwarden.RestoreBackupFile(fileName, passphrase, sessionKey, sleepMilliseconds, noLogout)
}

func BWGetSessionKey() (string, error) {
	if status, err := BWGetStatus(); err == nil {
		if status.Status == "unlocked" {
			if sessionKey, err := BWGetSessionKeyFromEnv(); err == nil {
				return sessionKey, nil
			}
		}
	}

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
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Println("An error occurred: ", err)
	}
	cmd.Wait()
	sessionKey, err := portwarden.ExtractSessionKey(stdout.String())
	if err != nil {
		return "", errors.New(string(stderr.Bytes()))
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

func BWGetStatus() (portwarden.Status, error) {
	status := portwarden.Status{}

	var cmd *exec.Cmd
	cmd = exec.Command("bw", "status")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return status, err
	}

	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		return status, err
	}

	return status, nil
}

func BWGetSessionKeyFromEnv() (string, error) {
	sessionKey := os.Getenv("BW_SESSION")
	if len(sessionKey) == 0 {
		return "", errors.New(ErrSessionKeyGetEnvFailed)
	}
	return sessionKey, nil
}
