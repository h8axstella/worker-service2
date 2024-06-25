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
		fmt.Println("Usage: worker-service <start-date> <end-date> [<worker-name>] [--worker-hash-only]")
		os.Exit(1)
	}

	startDate := os.Args[1]
	endDate := os.Args[2]
	var workerName string
	processWorkerHashOnly := false

	if len(os.Args) > 3 {
		for _, arg := range os.Args[3:] {
			if arg == "--worker-hash-only" {
				processWorkerHashOnly = true
			} else {
				workerName = arg
			}
		}
	}

	worker.ProcessWorkers(workerName, startDate, endDate, processWorkerHashOnly)
}
