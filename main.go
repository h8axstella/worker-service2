package main

import (
	"log"
	"time"
	"worker-service/config"
	"worker-service/database"
	"worker-service/worker"
)

func main() {
	config.InitConfig()

	database.Init()

	startWorkerProcessor()
}

func startWorkerProcessor() {
	ticker := time.NewTicker(24 * time.Hour)
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
