package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/tidwall/pretty"
	"github.com/vwxyzjn/portwarden"
)

const (
	BackupFolderName = "./portwarden_backup/"
	ErrVaultIsLocked = "Vault is locked."
)

func main() {
	pwes := []portwarden.PortWardenElement{}
	sessionKey := BWUnlockVaultToGetSessionKey()

	// save formmated json to "main.json"
	rawByte := BWListItemsRawBytes(sessionKey)
	if err := json.Unmarshal(rawByte, &pwes); err != nil {
		panic(err)
	}
	formattedByte := pretty.Pretty(rawByte)
	if err := ioutil.WriteFile(BackupFolderName+"main.json", formattedByte, 0644); err != nil {
		panic(err)
	}

	err := BWGetAllAttachments(BackupFolderName, sessionKey, pwes[:5])
	if err != nil {
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

func BWUnlockVaultToGetSessionKey() string {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := exec.Command("bw", "unlock")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	w.Close()
	out, _ := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	os.Stdout = rescueStdout
	return extractSessionKey(string(out))
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
