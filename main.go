package main

import (
	"fmt"
	"os"
	"worker-service/config"
	"worker-service/database"
	"worker-service/worker"
)

func main() {
	config.InitConfig()
	database.Init()

	if len(os.Args) < 3 {
		fmt.Println("Usage: worker-service <start-date> <end-date> [<worker-name>]")
		os.Exit(1)
	}

	startDate := os.Args[1]
	endDate := os.Args[2]
	var workerName string
	if len(os.Args) > 3 {
		workerName = os.Args[3]
	}

	worker.ProcessWorkers(workerName, startDate, endDate)
}
