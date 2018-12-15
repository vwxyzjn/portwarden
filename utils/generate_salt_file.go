package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const (
	Template = `package portwarden

const (
	Salt = "%v"
)
`
)

func main() {
	Salt := os.Getenv("Salt")
	if len(Salt) == 0 {
		log.Fatal("Salt not detected in Environment Variable `Salt`")
	}
	err := ioutil.WriteFile("./salt.go", []byte(fmt.Sprintf(Template, Salt)), 0644)
	if err != nil {
		if len(Salt) == 0 {
			log.Fatalf("Error writing salt file: %v", err)
		}
	}
}
