package worker

import (
	"fmt"
	"log"
	"worker-service/api"
	"worker-service/database"
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
			fmt.Printf("Using API Key: [REDACTED] for worker %s\n", worker.WorkerName)
			coins, err := api.FetchCoins(worker.AKey)
			if err != nil {
				log.Printf("Error fetching coins for worker %s: %v\n", worker.WorkerName, err)
				continue
			}
			fmt.Printf("Fetched coins for worker %s: %v\n", worker.WorkerName, coins)
			err = api.FetchHashrate(worker.AKey, worker.WorkerName, coins, worker.ID)
			if err != nil {
				log.Printf("Error fetching hashrate for worker %s: %v\n", worker.WorkerName, err)
				continue
			}
		}
	}
}
