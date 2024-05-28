package worker

import (
	"fmt"
	"log"
	"time"
	"worker-service/api"
	"worker-service/database"
	"worker-service/models"
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
					log.Printf("Error fetching hashrate for worker %s и coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}
			}

		case "emcd":
			fmt.Printf("Worker %s belongs to pool %s\n", worker.WorkerName, pool.PoolName)

			for _, coin := range coins {
				workersInfo, err := api.GetEmcdWorkersInfo(akey, coin)
				if err != nil {
					log.Printf("Error fetching workers info for worker %s и coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}

				log.Printf("\n%s Workers Info:\n", coin)
				log.Printf("Total Workers: %d (Active: %d, Inactive: %d)\n",
					workersInfo.TotalCount.All, workersInfo.TotalCount.Active, workersInfo.TotalCount.Inactive)
				log.Printf("Total Hashrate: %f\n", workersInfo.TotalHashrate.Hashrate)
				log.Printf("Total Hashrate (1h): %f\n", workersInfo.TotalHashrate.Hashrate1h)
				log.Printf("Total Hashrate (24h): %f\n", workersInfo.TotalHashrate.Hashrate24h)

				for _, detail := range workersInfo.Details {
					log.Printf("Worker: %s, Hashrate: %f, Hashrate (1h): %f, Hashrate (24h): %f, Active: %d\n",
						detail.Worker, detail.Hashrate, detail.Hashrate1h, detail.Hashrate24h, detail.Active)

					dailyHashInt := int64(detail.Hashrate24h)

					log.Printf("Inserting worker hash for worker %s\n", detail.Worker)
					workerHash := models.WorkerHash{
						FkWorker:   worker.ID,
						DailyHash:  dailyHashInt,
						HashDate:   time.Now(),
						FkPoolCoin: pool.ID,
					}
					err = database.UpdateWorkerHashrate(workerHash)
					if err != nil {
						log.Printf("Error updating worker hashrate for worker %s: %v\n", detail.Worker, err)
					}

					log.Printf("Inserting host hash for host %s\n", detail.Worker)
					hostHash := models.HostHash{
						FkHost:     worker.ID,
						DailyHash:  dailyHashInt,
						HashDate:   time.Now(),
						FkPoolCoin: pool.ID,
					}
					err = database.UpdateHostHashrate(hostHash)
					if err != nil {
						log.Printf("Error updating host hashrate for host %s: %v\n", detail.Worker, err)
					}
				}
			}
		}
	}
}
