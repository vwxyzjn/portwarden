package main

import (
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/vwxyzjn/portwarden"
)

func main() {
	var cnf = &config.Config{
		Broker:        "amqp://guest:guest@localhost:5672/",
		DefaultQueue:  "machinery_tasks",
		ResultBackend: "amqp://guest:guest@localhost:5672/",
		AMQP: &config.AMQPConfig{
			Exchange:     "machinery_exchange",
			ExchangeType: "direct",
			BindingKey:   "machinery_task",
		},
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		panic(err)
	}
	server.RegisterTasks(map[string]interface{}{
		"CreateBackupBytes": portwarden.CreateBackupBytes,
	})
	worker := server.NewWorker("worker_name", 0)
	err = worker.Launch()
	if err != nil {
		panic(err)
	}
}
