package worker

import (
	"fmt"
	"log"
	"worker-service/api"
	"worker-service/database"
	"worker-service/utils" // Для использования CreateSignature
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

		akey, skey, err := database.GetWorkerKeys(worker.ID)
		if err != nil {
			log.Printf("Error fetching keys for worker %s: %v\n", worker.WorkerName, err)
			continue
		}

		fmt.Printf("Using AKey:[REDACTED] for worker %s\n", worker.WorkerName)

		if pool.PoolName == "viabtc" {
			fmt.Printf("Worker %s belongs to pool %s\n", worker.WorkerName, pool.PoolName)

			fmt.Printf("Using SKey for signature: [REDACTED]\n")
			signature := utils.CreateSignature(skey, "GET", "/res/openapi/v1/hashrate/worker", "params")
			log.Printf("Signature created for worker %s: %s\n", worker.WorkerName, signature)

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
		} else {

			fmt.Printf("Worker %s belongs to pool %s, which does not use SKey\n", worker.WorkerName, pool.PoolName)

		}
	}
}
