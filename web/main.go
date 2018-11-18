package main

import (
	"github.com/vwxyzjn/portwarden/web/server"
)

func main() {
	ps := server.PortwardenServer{
		Port: 5000,
	}
	ps.Run()
}
