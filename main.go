package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"worker-service/auth"
	"worker-service/config"
	"worker-service/database"
	"worker-service/handlers"
	"worker-service/worker"
)

func main() {
	config.InitConfig()

	// Проверка подключения к БД (без вывода паролей)
	fmt.Printf("DB_HOST: %s\n", config.AppConfig.DBHost)
	fmt.Printf("DB_PORT: %s\n", config.AppConfig.DBPort)
	fmt.Printf("DB_USER: %s\n", config.AppConfig.DBUser)
	fmt.Printf("DB_NAME: %s\n", config.AppConfig.DBName)
	fmt.Printf("DB_SSLMODE: %s\n", config.AppConfig.DBSSLMode)
	fmt.Printf("APP_PORT: %s\n", config.AppConfig.Port)

	database.Init()

	go worker.StartWorkerProcessor()

	mux := http.NewServeMux()
	mux.Handle("/workers", auth.JWTMiddleware(http.HandlerFunc(handlers.WorkerHandler)))

	port := config.AppConfig.Port
	if port == "" {
		port = "8080"
	}

	certFile := "path/to/cert.pem"
	keyFile := "path/to/key.pem"

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}
