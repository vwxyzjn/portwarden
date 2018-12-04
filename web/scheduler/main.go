package main

import (
	"github.com/vwxyzjn/portwarden/web"
	"github.com/vwxyzjn/portwarden/web/scheduler/server"
)

func main() {
	web.InitCommonVars()
	ps := server.PortwardenServer{
		Port: 5000,
	}
	ps.Run()
}
