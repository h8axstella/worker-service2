package config

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	Port       string
}

var AppConfig Config

func InitConfig() {
	loadEnv()

	AppConfig = Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),
		Port:       os.Getenv("APP_PORT"),
	}

	if AppConfig.DBHost == "" || AppConfig.DBPort == "" || AppConfig.DBUser == "" || AppConfig.DBPassword == "" || AppConfig.DBName == "" || AppConfig.DBSSLMode == "" || AppConfig.Port == "" {
		log.Fatalf("Please ensure all required environment variables are set.")
	}
}

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && line[0] != '#' {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading .env file: %v", err)
	}
}
