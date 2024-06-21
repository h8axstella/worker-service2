package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	InfoLogger     *log.Logger
	WarningLogger  *log.Logger
	ErrorLogger    *log.Logger
	DurationLogger *log.Logger
)

func Init(logDir string) {
	err := os.MkdirAll(logDir, 0750)
	if err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	dateSuffix := time.Now().Format("2006-01-02")
	infoLogFile, err := os.OpenFile(filepath.Join(logDir, "info_"+dateSuffix+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		log.Fatalf("Failed to open info log file: %v", err)
	}
	InfoLogger = log.New(io.MultiWriter(os.Stdout, infoLogFile), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	warningLogFile, err := os.OpenFile(filepath.Join(logDir, "warning_"+dateSuffix+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		log.Fatalf("Failed to open warning log file: %v", err)
	}
	WarningLogger = log.New(io.MultiWriter(os.Stdout, warningLogFile), "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)

	errorLogFile, err := os.OpenFile(filepath.Join(logDir, "error_"+dateSuffix+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		log.Fatalf("Failed to open error log file: %v", err)
	}
	ErrorLogger = log.New(io.MultiWriter(os.Stdout, errorLogFile), "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	durationLogFile, err := os.OpenFile(filepath.Join(logDir, "duration_"+dateSuffix+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		log.Fatalf("Failed to open duration log file: %v", err)
	}
	DurationLogger = log.New(io.MultiWriter(os.Stdout, durationLogFile), "DURATION: ", log.Ldate|log.Ltime)
}

func LogInfo(message string) {
	InfoLogger.Println(message)
}

func LogWarning(message string) {
	WarningLogger.Println(message)
}

func LogError(message string) {
	ErrorLogger.Println(message)
}

func LogDuration(message string) {
	DurationLogger.Println(message)
}
