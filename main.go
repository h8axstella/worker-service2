// main.go
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

	if len(os.Args) >= 4 {
		workerName := os.Args[1]
		startDate := os.Args[2]
		endDate := os.Args[3]
		worker.ProcessWorkers(workerName, startDate, endDate)
	} else {
		fmt.Println("Usage: worker-service <worker-name> <start-date> <end-date>")
		os.Exit(1)
	}
}
