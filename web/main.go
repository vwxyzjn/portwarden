package main

import (
	"io/ioutil"
	"log"

	"github.com/vwxyzjn/portwarden/web/scheduler/server"
)

func main() {
	credential, err := ioutil.ReadFile("portwardenCredentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	ps := server.PortwardenServer{
		Port: 5000,
		GoogleDriveAppCredentials: credential,
	}
	ps.Run()
}
