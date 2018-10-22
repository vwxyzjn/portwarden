package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/vwxyzjn/portwarden"
)

const (
	ErrVaultIsLocked = "Vault is locked."
)

func main() {
	test := &portwarden.PortWardenElement{}
	fmt.Println(test)

	var stdout, stderr bytes.Buffer

	fmt.Println("Please enter your master password: (input is hidden)")
	cmd := exec.Command("bw", "unlock")
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println(extractSessionKey(string(stdout.Bytes())))
}

func extractSessionKey(stdout string) string {
	r := regexp.MustCompile(`BW_SESSION=".+"`)
	sessionKeyRawString := r.FindAllString(stdout, 1)[0]
	fmt.Println(sessionKeyRawString)
	sessionKey := strings.TrimPrefix(sessionKeyRawString, `BW_SESSION="`)
	sessionKey = sessionKey[:len(sessionKey)-1]
	return sessionKey
}
