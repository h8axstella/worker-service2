package worker

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
	"worker-service/api"
	"worker-service/database"
	"worker-service/models"
)

func StartWorkerProcessor() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	log.Printf("Starting initial worker processing at %s\n", time.Now())
	ProcessWorkers("2024-05-01", "2024-05-31")
	log.Printf("Initial worker processing completed at %s\n", time.Now())
	for {
		select {
		case <-ticker.C:
			log.Printf("Starting worker processing at %s\n", time.Now())
			ProcessWorkers("", "")
			log.Printf("Worker processing completed at %s\n", time.Now())
		}
	}
}

func ProcessWorkers(startDate, endDate string) {
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
					hashrateInt, err := strconv.ParseInt(history.Hashrate, 10, 64)
					if err != nil {
						log.Printf("Error converting hashrate to int64 for worker %s and coin %s: %v", worker.WorkerName, coin, err)
						continue
					}
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
							hashrateInt, err := strconv.ParseInt(history.Hashrate, 10, 64)
							if err != nil {
								log.Printf("Error converting hashrate to int64 for worker %s and coin %s: %v", host.WorkerName, coin, err)
								continue
							}
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
