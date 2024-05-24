package worker

import (
	"fmt"
	"log"
	"time"
	"worker-service/api"
	"worker-service/database"
)

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

		akey, _, err := database.GetWorkerKeys(worker.ID)
		if err != nil {
			log.Printf("Error fetching keys for worker %s: %v\n", worker.WorkerName, err)
			continue
		}

		fmt.Printf("Using AKey:[REDACTED] for worker %s\n", worker.WorkerName)

		coins, err := database.GetCoinsByPoolID(worker.FkPool)
		if err != nil {
			log.Printf("Error fetching coins for worker %s: %v\n", worker.WorkerName, err)
			continue
		}
		fmt.Printf("Fetched coins for worker %s: %v\n", worker.WorkerName, coins)

		switch pool.PoolName {
		case "viabtc":
			fmt.Printf("Worker %s belongs to pool %s\n", worker.WorkerName, pool.PoolName)

			for _, coin := range coins {
				err = api.FetchHashrate(pool.PoolURL, akey, worker.WorkerName, []string{coin}, worker.ID, pool.ID)
				if err != nil {
					log.Printf("Error fetching hashrate for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}
			}

			err = api.FetchAccountHashrate(pool.PoolURL, akey, coins, worker.ID, pool.ID)
			if err != nil {
				log.Printf("Error fetching account hashrate for worker %s: %v\n", worker.WorkerName, err)
				continue
			}

		case "f2pool":
			fmt.Printf("Worker %s belongs to pool %s\n", worker.WorkerName, pool.PoolName)

			for _, coin := range coins {
				err = api.FetchHashrate(pool.PoolURL, akey, worker.WorkerName, []string{coin}, worker.ID, pool.ID)
				if err != nil {
					log.Printf("Error fetching hashrate for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}
			}

		case "emcd":
			fmt.Printf("Worker %s belongs to pool %s\n", worker.WorkerName, pool.PoolName)

			for _, coin := range coins {
				err = api.FetchHashrate(pool.PoolURL, akey, worker.WorkerName, []string{coin}, worker.ID, pool.ID)
				if err != nil {
					log.Printf("Error fetching hashrate for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}
			}
		}
	}
}
