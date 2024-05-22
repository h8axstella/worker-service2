package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"worker-service/api"
	"worker-service/database"
	"worker-service/models"
)

func ProcessWorkers() {
	fmt.Println("Fetching active workers...")
	workers, err := database.GetActiveWorkers()
	if err != nil {
		log.Printf("Error fetching active workers: %v\n", err)
		return
	}

	for _, worker := range workers {
		fmt.Printf("Processing worker %s\n", worker.WorkerName)
		pool, err := database.GetPoolByID(worker.FkPool)
		if err != nil {
			log.Printf("Error fetching pool for worker %s: %v\n", worker.WorkerName, err)
			continue
		}

		if pool.PoolName == "viabtc" {
			fmt.Printf("Worker %s belongs to pool %s\n", worker.WorkerName, pool.PoolName)

			akey, skey, err := database.GetWorkerKeys(worker.ID)
			if err != nil {
				log.Printf("Error fetching keys for worker %s: %v\n", worker.WorkerName, err)
				continue
			}
			fmt.Printf("Using AKey: [REDACTED] and SKey: [REDACTED] for worker %s\n", worker.WorkerName)

			coins, err := api.FetchCoins(akey)
			if err != nil {
				log.Printf("Error fetching coins for worker %s: %v\n", worker.WorkerName, err)
				continue
			}
			fmt.Printf("Fetched coins for worker %s: %v\n", worker.WorkerName, coins)
			err = api.FetchHashrate(akey, worker.WorkerName, coins, worker.ID)
			if err != nil {
				log.Printf("Error fetching hashrate for worker %s: %v\n", worker.WorkerName, err)
				continue
			}
		}
	}
}

func StartWorkerProcessor() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	log.Printf("Starting initial worker processing at %s\n", time.Now())
	ProcessWorkers()
	log.Printf("Initial worker processing completed at %s\n", time.Now())

	for {
		select {
		case <-ticker.C:
			log.Printf("Starting worker processing at %s\n", time.Now())
			ProcessWorkers()
			log.Printf("Worker processing completed at %s\n", time.Now())
		}
	}
}

func GetWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := database.GetActiveWorkers()
	if err != nil {
		http.Error(w, "Error fetching workers", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(workers); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func CreateWorker(w http.ResponseWriter, r *http.Request) {
	var worker models.Worker
	if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	query := "INSERT INTO tb_worker (worker_name, akey, skey, fk_pool) VALUES ($1, $2, $3, $4) RETURNING id"
	err := database.DB.QueryRow(query, worker.WorkerName, worker.AKey, worker.SKey, worker.FkPool).Scan(&worker.ID)
	if err != nil {
		http.Error(w, "Failed to create worker", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(worker); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
