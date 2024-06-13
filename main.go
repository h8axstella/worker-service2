package main

import (
	"fmt"
	"time"
	"worker-service/common"
	"worker-service/config"
	"worker-service/course"
	"worker-service/database"
	"worker-service/logger"
	"worker-service/worker"
)

func main() {
	fmt.Println("Initializing config...")
	config.InitConfig()
	fmt.Println("Initializing database...")
	database.Init()
	fmt.Println("Database connected")

	logDir := "logs"
	fmt.Println("Initializing logger...")
	logger.Init(logDir)

	fmt.Println("Scheduling daily task...")
	go scheduleDailyTask()

	fmt.Println("Entering select statement...")
	select {}
}

func scheduleDailyTask() {
	fmt.Println("Calculating next run time...")
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)
	fmt.Printf("Next run time: %v\n", next)
	time.Sleep(time.Until(next))

	for {
		fmt.Println("Starting workers...")
		startWorkers()
		time.Sleep(24 * time.Hour)
	}
}

func startWorkers() {
	fmt.Println("Inside startWorkers function...")
	logger.InfoLogger.Println("Starting worker hashrate processing...")
	semaphore := make(chan struct{}, common.MaxConcurrentRequests)
	go func() {
		fmt.Println("Starting worker hashrate processor...")
		worker.StartWorkerHashrateProcessor(semaphore, common.MaxRetryAttempts)
	}()
	go func() {
		fmt.Println("Scheduling BTC processing...")
		course.ScheduleBTCProcessing()
	}()
}
