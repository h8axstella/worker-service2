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

	if len(os.Args) >= 3 {
		startDate := os.Args[1]
		endDate := os.Args[2]
		worker.ProcessWorkers(startDate, endDate)
	} else {
		fmt.Println("Usage: worker-service <start-date> <end-date>")
		os.Exit(1)
	}
}
