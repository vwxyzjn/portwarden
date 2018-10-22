package main

import (
	"fmt"
	"io/ioutil"
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

	realStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w

	cmd := exec.Command("bw", "unlock")
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	w.Close()

	out, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	os.Stdout = realStdout
	fmt.Println(extractSessionKey(string(out)))
}

func extractSessionKey(stdout string) string {
	r := regexp.MustCompile(`BW_SESSION=".+"`)
	sessionKeyRawString := r.FindAllString(stdout, 1)[0]
	fmt.Println(sessionKeyRawString)
	sessionKey := strings.TrimPrefix(sessionKeyRawString, `BW_SESSION="`)
	sessionKey = sessionKey[:len(sessionKey)-1]
	return sessionKey
}
