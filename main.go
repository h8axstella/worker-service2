package main

import (
	"time"
	"worker-service/common"
	"worker-service/config"
	"worker-service/course"
	"worker-service/database"
	"worker-service/logger"
	"worker-service/worker"
)

func main() {
	config.InitConfig()
	database.Init()

	logDir := "logs"
	logger.Init(logDir)

	go scheduleDailyTask()

	select {}
}

func scheduleDailyTask() {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)
	time.Sleep(time.Until(next))

	for {
		startWorkers()
		time.Sleep(24 * time.Hour)
	}
}

func startWorkers() {
	logger.InfoLogger.Println("Starting worker hashrate processing...")
	semaphore := make(chan struct{}, common.MaxConcurrentRequests)
	go worker.StartWorkerHashrateProcessor(semaphore, common.MaxRetryAttempts)
	go course.ScheduleBTCProcessing()
}
