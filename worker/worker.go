package worker

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
	"worker-service/api"
	"worker-service/database"
	"worker-service/models"
)

func ProcessWorkers(workerName, startDate, endDate string) {
	fmt.Println("Fetching active workers...")

	var workers []models.Worker
	var err error

	if workerName != "" {
		var worker models.Worker
		worker, err = database.GetWorkerByName(workerName)
		if err != nil {
			log.Printf("Error fetching worker %s: %v\n", workerName, err)
			return
		}
		if worker.AKey == "" || worker.FkPool == "" {
			log.Printf("Skipping worker %s due to missing akey or fk_pool", worker.WorkerName)
			return
		}
		workers = append(workers, worker)
	} else {
		workers, err = database.GetActiveWorkers()
		if err != nil {
			log.Printf("Error fetching active workers: %v\n", err)
			return
		}
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 1)

	for _, worker := range workers {
		if worker.AKey == "" || worker.FkPool == "" {
			log.Printf("Skipping worker %s due to missing akey или fk_pool", worker.WorkerName)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(worker models.Worker) {
			defer wg.Done()
			defer func() { <-semaphore }()
			fmt.Printf("Processing worker %s\n", worker.WorkerName)

			pool, err := database.GetPoolByID(worker.FkPool)
			if err != nil {
				log.Printf("Error fetching pool for worker %s: %v\n", worker.WorkerName, err)
				return
			}

			akey, _, err := database.GetWorkerKeys(worker.ID)
			if err != nil {
				log.Printf("Error fetching keys for worker %s: %v\n", worker.WorkerName, err)
				return
			}

			fmt.Printf("Using AKey:[REDACTED] for worker %s\n", worker.WorkerName)

			coins, err := database.GetCoinsByPoolID(worker.FkPool)
			if err != nil {
				log.Printf("Error fetching coins for worker %s: %v\n", worker.WorkerName, err)
				return
			}
			fmt.Printf("Fetched coins for worker %s: %v\n", worker.WorkerName, coins)

			for _, coin := range coins {
				asicList, err := api.FetchWorkerList(pool.PoolURL, akey, coin)
				if err != nil {
					log.Printf("Error fetching worker list for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}

				for _, asic := range asicList {
					host, err := database.GetHostByWorkerName(asic.WorkerName)
					if err != nil {
						log.Printf("WorkerName %s does not match any device of account %s, saving to unident hash", asic.WorkerName, worker.WorkerName)
						// Save to tb_unident_hash
						saveToUnidentHash(worker.ID, worker.FkPool, coin, asic, pool.PoolURL, akey, startDate, endDate)
						continue
					}

					poolCoinUUID, err := database.GetPoolCoinUUID(worker.FkPool, coin)
					if err != nil {
						log.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", worker.FkPool, coin, err)
						continue
					}
					if poolCoinUUID == "" {
						log.Printf("PoolCoinUUID is empty for pool %s and coin %s", worker.FkPool, coin)
						continue
					}

					workerHistory, err := api.FetchWorkerHashrateHistory(pool.PoolURL, akey, asic.WorkerID, coin, startDate, endDate)
					if err != nil {
						log.Printf("Error fetching worker hashrate history for worker %s and coin %s: %v\n", asic.WorkerName, coin, err)
						continue
					}

					for _, history := range workerHistory {
						dailyHashFloat, err := strconv.ParseFloat(history.Hashrate, 64)
						if err != nil {
							log.Printf("Error converting hashrate to float64 for worker %s and coin %s: %v", asic.WorkerName, coin, err)
							continue
						}

						hostHash := models.HostHash{
							FkHost:       host.ID,
							FkPoolCoin:   poolCoinUUID,
							DailyHash:    dailyHashFloat,
							HashDate:     history.Date,
							FkPool:       worker.FkPool,
							HostWorkerID: strconv.Itoa(asic.WorkerID),
						}

						err = database.UpdateHostHashrate(hostHash)
						if err != nil {
							log.Printf("Error updating hashrate for host %s: %v", hostHash.FkHost, err)
							continue
						}
						log.Printf("Successfully updated hashrate for host %s on date %s", hostHash.FkHost, hostHash.HashDate)
					}
				}
			}
		}(worker)
	}

	wg.Wait()
	fmt.Println("All workers processed")
}

func saveToUnidentHash(workerID, poolID, coin string, asic models.WorkerListItem, poolURL, akey, startDate, endDate string) {
	poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
	if err != nil {
		log.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", poolID, coin, err)
		return
	}
	if poolCoinUUID == "" {
		log.Printf("PoolCoinUUID is empty for pool %s and coin %s", poolID, coin)
		return
	}

	workerHistory, err := api.FetchWorkerHashrateHistory(poolURL, akey, asic.WorkerID, coin, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching worker hashrate history for worker %s and coin %s: %v\n", asic.WorkerName, coin, err)
		return
	}

	for _, history := range workerHistory {
		dailyHashFloat, err := strconv.ParseFloat(history.Hashrate, 64)
		if err != nil {
			log.Printf("Error converting hashrate to float64 for worker %s and coin %s: %v", asic.WorkerName, coin, err)
			continue
		}

		unidentHash := models.UnidentHash{
			HashDate:     history.Date,
			DailyHash:    int64(dailyHashFloat),
			HostWorkerID: strconv.Itoa(asic.WorkerID),
			UnidentName:  asic.WorkerName,
			FkWorker:     workerID,
			FkPoolCoin:   poolCoinUUID,
			LastEdit:     time.Now().Format("2006-01-02"),
			Status:       0,
		}

		err = database.InsertUnidentHash(unidentHash)
		if err != nil {
			log.Printf("Error inserting unident hashrate for worker %s: %v", unidentHash.UnidentName, err)
			continue
		}
		log.Printf("Successfully inserted unident hashrate for worker %s on date %s", unidentHash.UnidentName, unidentHash.HashDate)
	}
}
