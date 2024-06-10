package main

import (
	"worker-service/config"
	"worker-service/course"
	"worker-service/database"
	"worker-service/worker"
)

func main() {
	config.InitConfig()
	database.Init()
	go worker.StartWorkerHashrateProcessor()
	go course.ScheduleBTCProcessing()
	select {}
}
