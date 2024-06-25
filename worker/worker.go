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

func ProcessWorkers(workerName, startDate, endDate string, processWorkerHashOnly bool) {
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
				if startDate != "" && endDate != "" {
					accountHistory, err := api.FetchAccountHashrateHistory(pool.PoolURL, akey, coin, startDate, endDate)
					if err != nil {
						log.Printf("Error fetching account hashrate history for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
						continue
					}
					if len(accountHistory) == 0 {
						log.Printf("No account hashrate history found for worker %s and coin %s", worker.WorkerName, coin)
						continue
					}
					log.Printf("Fetched account hashrate history for worker %s and coin %s: %+v\n", worker.WorkerName, coin, accountHistory)

					for _, history := range accountHistory {
						poolCoinID, err := database.GetPoolCoinUUID(worker.FkPool, coin)
						if err != nil {
							log.Printf("Error fetching PoolCoinID for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
							continue
						}
						hashrateFloat, err := strconv.ParseFloat(history.Hashrate, 64)
						if err != nil {
							log.Printf("Error converting hashrate to float64 for worker %s and coin %s: %v", worker.WorkerName, coin, err)
							continue
						}

						workerHash := models.WorkerHash{
							FkWorker:   worker.ID,
							FkPoolCoin: poolCoinID,
							DailyHash:  hashrateFloat,
							HashDate:   history.Date,
							FkPool:     worker.FkPool,
						}

						log.Printf("Creating workerHash: %+v\n", workerHash)
						err = database.UpdateWorkerHashrate(workerHash)
						if err != nil {
							log.Printf("Error updating account hashrate history for worker %s: %v\n", worker.WorkerName, err)
						} else {
							log.Printf("Successfully updated account hashrate history for worker %s on date %s\n", worker.WorkerName, history.Date)
						}
					}
				} else {
					err = api.FetchHashrate(pool.PoolURL, akey, worker.WorkerName, []string{coin}, worker.ID, pool.ID)
					if err != nil {
						log.Printf("Error fetching hashrate for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
						continue
					}
				}

				if !processWorkerHashOnly {
					// Обработка хостов
					hosts, err := database.GetHostsByWorkerID(worker.ID)
					if err != nil {
						log.Printf("Error fetching hosts for worker %s: %v", worker.WorkerName, err)
						continue
					}

					asicList, err := api.FetchWorkerList(pool.PoolURL, akey, coin)
					if err != nil {
						log.Printf("Error fetching worker list for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
						continue
					}

					for _, asic := range asicList {
						foundHost := false
						for _, host := range hosts {
							if asic.WorkerName == host.WorkerName {
								log.Printf("Updating host worker_id for host ID %s with worker ID %d", host.ID, asic.WorkerID)
								err := database.UpdateHostWorkerID(asic.WorkerID, host.ID)
								if err != nil {
									log.Printf("Error updating worker_id for host %s: %v\n", host.WorkerName, err)
								}
								foundHost = true
							}
						}
						if !foundHost {
							saveToUnidentHash(worker.ID, worker.FkPool, coin, asic, pool.PoolURL, akey, startDate, endDate)
						}
					}

					for _, host := range hosts {
						hostHashes := []models.HostHash{}
						if host.HostWorkerID.Valid {
							workerHistory, err := api.FetchWorkerHashrateHistory(pool.PoolURL, akey, int(host.HostWorkerID.Int64), coin, startDate, endDate)
							if err != nil {
								log.Printf("Error fetching worker hashrate history for worker %s and coin %s: %v\n", host.WorkerName, coin, err)
								continue
							}
							for _, history := range workerHistory {
								poolCoinID, err := database.GetPoolCoinUUID(worker.FkPool, coin)
								if err != nil {
									log.Printf("Error fetching PoolCoinID for worker %s and coin %s: %v\n", host.WorkerName, coin, err)
									continue
								}
								hashrateFloat, err := strconv.ParseFloat(history.Hashrate, 64)
								if err != nil {
									log.Printf("Error converting hashrate to float64 for worker %s and coin %s: %v", host.WorkerName, coin, err)
									continue
								}

								hostHash := models.HostHash{
									FkHost:       host.ID,
									FkPoolCoin:   poolCoinID,
									DailyHash:    hashrateFloat,
									HashDate:     history.Date,
									FkPool:       worker.FkPool,
									HostWorkerID: strconv.Itoa(int(host.HostWorkerID.Int64)), // Преобразование в строку
								}
								hostHashes = append(hostHashes, hostHash)
							}
						}
						for _, hostHash := range hostHashes {
							err := database.UpdateHostHashrate(hostHash)
							if err != nil {
								log.Printf("Error updating worker hashrate history for worker %s: %v\n", host.WorkerName, err)
							} else {
								log.Printf("Successfully updated worker hashrate history for worker %s on date %s\n", host.WorkerName, hostHash.HashDate)
							}
						}
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
