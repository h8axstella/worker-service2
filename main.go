package main

import (
	"log"
	"time"

	"worker-service/config"
	"worker-service/course"
	"worker-service/database"
	"worker-service/worker"
)

func main() {
	config.InitConfig()
	database.Init()
	go startWorkerProcessor()
	go scheduleBTCProcessing()
	select {}
}

func startWorkerProcessor() {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	log.Printf("Starting initial worker processing at %s\n", time.Now())
	worker.ProcessWorkers()
	log.Printf("Initial worker processing completed at %s\n", time.Now())
	for {
		select {
		case <-ticker.C:
			log.Printf("Starting worker processing at %s\n", time.Now())
			worker.ProcessWorkers()
			log.Printf("Worker processing completed at %s\n", time.Now())
		}
	}
}

// Исправленная функция scheduleBTCProcessing
func scheduleBTCProcessing() {
	// First run immediately
	course.ProcessBTCPrice()

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Println("Scheduled BTC price processing started.")
			course.ProcessBTCPrice()
		}
	}
}
