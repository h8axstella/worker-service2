package main

import (
	"fmt"
	"time"
	"worker-service/config"
	"worker-service/course"
	"worker-service/database"
	"worker-service/logger"
	"worker-service/worker"
)

func main() {
	logDir := "logs"
	fmt.Println("Initializing logger...")
	logger.Init(logDir)
	logger.LogInfo("Logger initialized")

	fmt.Println("Initializing config...")
	logger.LogInfo("Initializing config...")
	config.InitConfig()

	fmt.Println("Initializing database...")
	logger.LogInfo("Initializing database...")
	database.Init()
	defer func() {
		logger.LogInfo("Closing database...")
		database.Close()
	}()
	fmt.Println("Database connected")
	logger.LogInfo("Database connected")

	fmt.Println("Scheduling daily task...")
	logger.LogInfo("Scheduling daily task...")
	go scheduleDailyTask()

	fmt.Println("Starting initial task...")
	logger.LogInfo("Starting initial task...")
	startWorkers()

	fmt.Println("Entering select statement...")
	logger.LogInfo("Entering select statement...")
	select {}
}

func scheduleDailyTask() {
	fmt.Println("Calculating next run time...")
	logger.LogInfo("Calculating next run time...")
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)
	fmt.Printf("Next run time: %v\n", next)
	logger.LogInfo(fmt.Sprintf("Next run time: %v", next))
	time.Sleep(time.Until(next))

	for {
		fmt.Println("Starting workers...")
		logger.LogInfo("Starting workers...")
		startWorkers()
		time.Sleep(24 * time.Hour)
	}
}

func startWorkers() {
	startTime := time.Now()
	fmt.Printf("Task started at: %v\n", startTime)
	logger.LogInfo(fmt.Sprintf("Task started at: %v", startTime))
	logger.LogDuration(fmt.Sprintf("Task started at: %v", startTime))

	fmt.Println("Inside startWorkers function...")
	logger.LogInfo("Inside startWorkers function...")
	logger.LogInfo("Starting worker hashrate processing...")
	apiSemaphore := make(chan struct{}, 5)
	dbSemaphore := make(chan struct{}, 20)
	done := make(chan bool)

	go func() {
		fmt.Println("Starting worker hashrate processor...")
		logger.LogInfo("Starting worker hashrate processor...")
		worker.StartWorkerHashrateProcessor(apiSemaphore, dbSemaphore, 3)
		done <- true
	}()
	go func() {
		fmt.Println("Scheduling BTC processing...")
		logger.LogInfo("Scheduling BTC processing...")
		course.ScheduleBTCProcessing()
		done <- true
	}()

	// Wait for both tasks to complete
	select {
	case <-done:
		logger.LogInfo("Worker hashrate processor completed.")
	case <-done:
		logger.LogInfo("BTC processing completed.")
	}

	endTime := time.Now()
	fmt.Printf("Task ended at: %v\n", endTime)
	logger.LogInfo(fmt.Sprintf("Task ended at: %v", endTime))
	logger.LogDuration(fmt.Sprintf("Task ended at: %v", endTime))

	duration := endTime.Sub(startTime)
	fmt.Printf("Task duration: %v\n", duration)
	logger.LogInfo(fmt.Sprintf("Task duration: %v", duration))
	logger.LogDuration(fmt.Sprintf("Task duration: %v", duration))
}
