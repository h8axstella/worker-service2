package worker

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
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

	for _, worker := range workers {
		if worker.AKey == "" || worker.FkPool == "" {
			log.Printf("Skipping worker %s due to missing akey or fk_pool", worker.WorkerName)
			continue
		}

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

		for _, coin := range coins {
			if startDate != "" && endDate != "" {
				accountHistory, err := api.FetchAccountHashrateHistory(pool.PoolURL, akey, coin, startDate, endDate)
				if err != nil {
					log.Printf("Error fetching account hashrate history for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}
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
					hashrateInt := int64(hashrateFloat)
					accountHash := models.WorkerHash{
						FkWorker:   worker.ID,
						FkPoolCoin: poolCoinID,
						DailyHash:  hashrateInt,
						HashDate:   history.Date,
					}
					err = database.UpdateWorkerHashrate(accountHash)
					if err != nil {
						log.Printf("Error updating account hashrate history for worker %s: %v\n", worker.WorkerName, err)
					} else {
						log.Printf("Successfully updated account hashrate history for worker %s on date %s\n", worker.WorkerName, history.Date)
					}
				}

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
					for _, host := range hosts {
						if asic.WorkerName == host.WorkerName {
							host.HostWorkerID = sql.NullInt64{Int64: int64(asic.WorkerID), Valid: true}
							err := database.UpdateHostWorkerID(int(host.HostWorkerID.Int64), host.ID)
							if err != nil {
								log.Printf("Error updating worker_id for host %s: %v\n", host.WorkerName, err)
							}
						}
					}
				}

				for _, host := range hosts {
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
							hashrateInt := int64(hashrateFloat) // Преобразование float64 в int64
							hostHash := models.HostHash{
								FkHost:     host.ID,
								FkPoolCoin: poolCoinID,
								DailyHash:  hashrateInt,
								HashDate:   history.Date,
							}
							err = database.UpdateHostHashrate(hostHash)
							if err != nil {
								log.Printf("Error updating worker hashrate history for worker %s: %v\n", host.WorkerName, err)
							} else {
								log.Printf("Successfully updated worker hashrate history for worker %s on date %s\n", host.WorkerName, history.Date)
							}
						}
					}
				}
			} else {
				err = api.FetchHashrate(pool.PoolURL, akey, worker.WorkerName, []string{coin}, worker.ID, pool.ID)
				if err != nil {
					log.Printf("Error fetching hashrate for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
					continue
				}
			}
		}
	}
}
