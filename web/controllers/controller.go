package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vwxyzjn/portwarden"
	melody "gopkg.in/olahol/melody.v1"
)

func EncryptBackupController(c *gin.Context) {
	m := melody.New()
	m.HandleRequest(c.Writer, c.Request)
	m.HandleConnect(func(s *melody.Session) {
		_, err := BWGetSessionKey(m)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		m.Broadcast([]byte("you know what"))
		// portwarden.EncryptBackup(fileName, passphrase, sessionKey, sleepMilliseconds)
	})
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast([]byte("you know what"))
	})
}

func DecryptBackupController(m *melody.Melody, msg []byte) {
	// return portwarden.DecryptBackup(fileName, passphrase)
}

func BWGetSessionKey(m *melody.Melody) (string, error) {
	sessionKey, err := BWUnlockVaultToGetSessionKey(m)
	if err != nil {
		if err.Error() == portwarden.BWErrNotLoggedIn {
			sessionKey, err = BWLoginGetSessionKey(m)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return sessionKey, err
}

func BWUnlockVaultToGetSessionKey(m *melody.Melody) (string, error) {
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

func BWLoginGetSessionKey(m *melody.Melody) (string, error) {
	cmd := exec.Command("bw", "login")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return "", err
	}
	m.Broadcast(stderr.Bytes())

	go func() {
		time.Sleep(time.Millisecond * 200)
		m.Broadcast(stderr.Bytes())
	}()

	cmd.Wait()
	sessionKey, err := portwarden.ExtractSessionKey(stdout.String())
	if err != nil {
		return "", errors.New(string(stdout.Bytes()))
	}
	return sessionKey, nil
}
