package main

import (
	"os"
	"worker-service/config"
	"worker-service/course"
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
		go worker.StartWorkerProcessor()
		go course.ScheduleBTCProcessing()
		select {}
	}
}
