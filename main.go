package main

import (
	"fmt"
	"log"
	"net/http"
	"worker-service/config"
	"worker-service/database"
	"worker-service/handlers"
	"worker-service/worker"
)

func main() {
	config.InitConfig()

	fmt.Printf("DB_HOST: %s\n", config.AppConfig.DBHost)
	fmt.Printf("DB_PORT: %s\n", config.AppConfig.DBPort)
	fmt.Printf("DB_USER: %s\n", config.AppConfig.DBUser)
	fmt.Printf("DB_NAME: %s\n", config.AppConfig.DBName)
	fmt.Printf("DB_SSLMODE: %s\n", config.AppConfig.DBSSLMode)
	fmt.Printf("APP_PORT: %s\n", config.AppConfig.Port)

	database.Init()

	go worker.StartWorkerProcessor()

	http.HandleFunc("/workers", handlers.WorkerHandler)
	port := config.AppConfig.Port
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
