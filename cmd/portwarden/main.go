package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/vwxyzjn/portwarden"
)

const (
	ErrVaultIsLocked = "Vault is locked."
)

func main() {
	test := []portwarden.PortWardenElement{}
	fmt.Println(test)

	sessionKey := BWUnlockVaultToGetSessionKey()
	rawByte := BWListItemsRawBytes(sessionKey)
	if err := json.Unmarshal(rawByte, &test); err != nil {
		panic(err)
	}
	spew.Dump(test[:5])
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
