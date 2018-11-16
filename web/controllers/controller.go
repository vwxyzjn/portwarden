package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/vwxyzjn/portwarden"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func EncryptBackupController(c *gin.Context) {
	conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil) // error ignored for sake of simplicity

	go func() {
		for {
			// Read message from browser
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Print the message to the console
			fmt.Printf("%s read:\n %d, %s", conn.RemoteAddr(), msgType, msg)
		}
	}()

	_, err := BWGetSessionKey(conn)
	if err != nil {
		panic(err)
	}
}

func DecryptBackupController(c *websocket.Conn, msg []byte) {
	// return portwarden.DecryptBackup(fileName, passphrase)
}

func BWGetSessionKey(c *websocket.Conn) (string, error) {
	sessionKey, err := BWUnlockVaultToGetSessionKey(c)
	if err != nil {
		if err.Error() == portwarden.BWErrNotLoggedIn {
			sessionKey, err = BWLoginGetSessionKey(c)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return sessionKey, err
}

func BWUnlockVaultToGetSessionKey(c *websocket.Conn) (string, error) {
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

func BWLoginGetSessionKey(c *websocket.Conn) (string, error) {
	cmd := exec.Command("bw", "login")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return "", err
	}

	go func() {
		time.Sleep(time.Millisecond * 500)
		if err := c.WriteMessage(1, stderr.Bytes()); err != nil {
		}
	}()

	cmd.Wait()
	sessionKey, err := portwarden.ExtractSessionKey(stdout.String())
	if err != nil {
		return "", errors.New(string(stdout.Bytes()))
	}
	return sessionKey, nil
}
