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

			akey, skey, err := database.GetWorkerKeys(worker.ID)
			if err != nil {
				log.Printf("Error fetching keys for worker %s: %v\n", worker.WorkerName, err)
				continue
			}
			fmt.Printf("Using AKey: [REDACTED] and SKey: [REDACTED] for worker %s\n", worker.WorkerName)

			log.Printf("Fetched SKey: %s for worker %s (for debugging purposes)\n", skey, worker.WorkerName)

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
