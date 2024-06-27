package worker

import (
	"fmt"
	"log"
	"sync"
	"time"
	"worker-service/api"
	"worker-service/common"
	"worker-service/database"
	"worker-service/logger"
	"worker-service/models"
)

func StartWorkerHashrateProcessor(apiSemaphore, dbSemaphore chan struct{}, maxRetryAttempts int) {
	fmt.Println("Worker hashrate processor started...")
	logger.InfoLogger.Println("Worker hashrate processor started...")
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	log.Printf("Starting initial worker processing at %s\n", time.Now())
	logger.InfoLogger.Printf("Starting initial worker processing at %s\n", time.Now())
	ProcessActiveWorkers(apiSemaphore, dbSemaphore, maxRetryAttempts)
	log.Printf("Initial worker processing completed at %s\n", time.Now())
	logger.InfoLogger.Printf("Initial worker processing completed at %s\n", time.Now())
	for {
		select {
		case <-ticker.C:
			log.Printf("Starting worker processing at %s\n", time.Now())
			logger.InfoLogger.Printf("Starting worker processing at %s\n", time.Now())
			ProcessActiveWorkers(apiSemaphore, dbSemaphore, maxRetryAttempts)
			log.Printf("Worker processing completed at %s\n", time.Now())
			logger.InfoLogger.Printf("Worker processing completed at %s\n", time.Now())
		}
	}
}

func ProcessActiveWorkers(apiSemaphore, dbSemaphore chan struct{}, maxRetryAttempts int) {
	fmt.Println("Fetching active workers...")
	logger.InfoLogger.Println("Fetching active workers...")

	workers, err := database.GetActiveWorkers()
	if err != nil {
		log.Printf("Error fetching active workers: %v\n", err)
		logger.ErrorLogger.Printf("Error fetching active workers: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	for _, worker := range workers {
		wg.Add(1)
		go func(worker models.Worker) {
			defer wg.Done()
			apiSemaphore <- struct{}{}
			defer func() { <-apiSemaphore }()
			processWorker(worker, dbSemaphore, maxRetryAttempts)
		}(worker)
	}
	wg.Wait()
}

func processWorker(worker models.Worker, dbSemaphore chan struct{}, maxRetryAttempts int) {
	if worker.AKey == "" || worker.FkPool == "" {
		log.Printf("Skipping worker %s due to missing akey or fk_pool", worker.WorkerName)
		logger.WarningLogger.Printf("Skipping worker %s due to missing akey или fk_pool", worker.WorkerName)
		return
	}

	pool, err := database.GetPoolByID(worker.FkPool)
	if err != nil {
		log.Printf("Error fetching pool for worker %s: %v\n", worker.WorkerName, err)
		logger.ErrorLogger.Printf("Error fetching pool for worker %s: %v\n", worker.WorkerName, err)
		return
	}

	akey, skey, err := database.GetWorkerKeys(worker.ID)
	if err != nil {
		log.Printf("Error fetching keys for worker %s: %v\n", worker.WorkerName, err)
		logger.ErrorLogger.Printf("Error fetching keys for worker %s: %v\n", worker.WorkerName, err)
		return
	}

	baseURL := pool.PoolURL

	var coins []string
	if pool.PoolName == "f2pool" {
		coins, err = database.GetFullNameCoinsByPoolID(worker.FkPool)
	} else {
		coins, err = database.GetCoinsByPoolID(worker.FkPool)
	}

	if err != nil {
		log.Printf("Error fetching coins for worker %s: %v\n", worker.WorkerName, err)
		logger.ErrorLogger.Printf("Error fetching coins for worker %s: %v\n", worker.WorkerName, err)
		return
	}

	switch pool.PoolName {
	case "viabtc":
		processViaBTCHashrate(pool, worker, coins, akey, dbSemaphore, maxRetryAttempts)
	case "f2pool":
		processF2PoolHashrate(pool, worker, coins, akey, dbSemaphore, maxRetryAttempts)
	case "emcd":
		processEmcd(pool, worker, coins, akey, dbSemaphore, maxRetryAttempts)
	case "binance":
		processBinance(pool, worker, coins, akey, *skey, baseURL, dbSemaphore, maxRetryAttempts)
	}
}

func processBinance(pool models.Pool, worker models.Worker, coins []string, akey, skey, baseURL string, dbSemaphore chan struct{}, maxRetryAttempts int) {
	for _, coin := range coins {
		workersInfo, err := api.GetWorkerList(akey, skey, "sha256", worker.WorkerName, baseURL)
		if err != nil {
			log.Printf("Error fetching workers info for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
			continue
		}

		for _, workerInfo := range workersInfo {
			hosts, err := database.GetHostsByWorkerID(worker.ID)
			if err != nil {
				log.Printf("Error fetching hosts for account %s: %v", worker.WorkerName, err)
				continue
			}

			matchFound := false
			for _, host := range hosts {
				if workerInfo.HashRateInfo.Name == host.WorkerName {
					matchFound = true
					hostHash := models.HostHash{
						FkHost:     host.ID,
						DailyHash:  workerInfo.HashRateInfo.H24HashRate,
						HashDate:   time.Now(),
						FkPoolCoin: pool.ID,
						FkPool:     pool.ID,
					}
					dbSemaphore <- struct{}{}
					go func() {
						defer func() { <-dbSemaphore }()
						err = database.UpdateHostHashrate(hostHash, pool.ID)
						if err != nil {
							log.Printf("Error updating host hashrate for host %s: %v", workerInfo.HashRateInfo.Name, err)
						}
					}()
					break
				}
			}

			if !matchFound {
				unidentHash := models.UnidentHash{
					HashDate:     time.Now(),
					DailyHash:    workerInfo.HashRateInfo.H24HashRate,
					HostWorkerID: worker.ID,
					UnidentName:  workerInfo.HashRateInfo.Name,
					FkWorker:     worker.ID,
					FkPoolCoin:   pool.ID,
					FkPool:       pool.ID,
				}
				dbSemaphore <- struct{}{}
				go func() {
					defer func() { <-dbSemaphore }()
					err = database.InsertUnidentHash(unidentHash)
					if err != nil {
						log.Printf("Error inserting unident hash for worker %s: %v", workerInfo.HashRateInfo.Name, err)
					}
				}()
			}
		}

		// Обработка общего хешрейта аккаунта
		accountHash := models.WorkerHash{
			FkWorker:   worker.ID,
			FkPoolCoin: pool.ID,
			DailyHash:  workersInfo[0].HashRateInfo.H24HashRate, // Assuming the first entry represents 24-hour account hashrate
			HashDate:   time.Now(),
			FkPool:     pool.ID,
		}
		dbSemaphore <- struct{}{}
		go func() {
			defer func() { <-dbSemaphore }()
			err = database.UpdateWorkerHashrate(accountHash, pool.ID)
			if err != nil {
				log.Printf("Error updating account hashrate for worker %s: %v", worker.WorkerName, err)
			}
		}()
	}
}

func processViaBTCHashrate(pool models.Pool, worker models.Worker, coins []string, akey string, dbSemaphore chan struct{}, maxRetryAttempts int) {
	var wg sync.WaitGroup

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()
			err := common.Retry(maxRetryAttempts, 2, func() error {
				return api.FetchViaBTCWorkerHashrate(pool.PoolURL, akey, worker.WorkerName, []string{coin}, worker.ID, pool.ID)
			})
			if err != nil {
				log.Printf("Error fetching hashrate for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
				return
			}

			dbSemaphore <- struct{}{}
			go func() {
				defer func() { <-dbSemaphore }()
				err := common.Retry(maxRetryAttempts, 2, func() error {
					return api.FetchOverallAccountHashrate(pool.PoolURL, akey, coins, worker.ID, pool.ID)
				})
				if err != nil {
					log.Printf("Error fetching account hashrate for worker %s: %v\n", worker.WorkerName, err)
				}
			}()
		}(coin)
	}

	wg.Wait()
}

func processF2PoolHashrate(pool models.Pool, worker models.Worker, coins []string, akey string, dbSemaphore chan struct{}, maxRetryAttempts int) {
	var wg sync.WaitGroup

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()
			err := common.Retry(maxRetryAttempts, 2, func() error {
				workers, err := api.GetF2PoolWorkerHashrate(pool.PoolURL, akey, worker.WorkerName, coin)
				if err != nil {
					return err
				}
				return api.ProcessF2PoolWorkers(pool, worker, coin, workers, dbSemaphore)
			})
			if err != nil {
				log.Printf("Error fetching hashrate for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
				return
			}
		}(coin)
	}

	wg.Wait()
}

func processEmcd(pool models.Pool, worker models.Worker, coins []string, akey string, dbSemaphore chan struct{}, maxRetryAttempts int) {
	for _, coin := range coins {
		workersInfo, err := api.GetEmcdWorkersInfo(akey, coin, pool.PoolURL)
		if err != nil {
			log.Printf("Error fetching workers info for worker %s and coin %s: %v\n", worker.WorkerName, coin)
			continue
		}
		log.Printf("\n%s Workers Info:\n", coin)
		log.Printf("Total Workers: %d (Active: %d, Inactive: %d)\n", workersInfo.TotalCount.All, workersInfo.TotalCount.Active, workersInfo.TotalCount.Inactive)
		log.Printf("Total Hashrate: %f\n", workersInfo.TotalHashrate.Hashrate)
		log.Printf("Total Hashrate (1h): %f\n", workersInfo.TotalHashrate.Hashrate1h)
		log.Printf("Total Hashrate (24h): %f\n", workersInfo.TotalHashrate.Hashrate24h)

		poolCoinID, err := database.GetPoolCoinUUID(pool.ID, coin)
		if err != nil {
			log.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v\n", pool.ID, coin, err)
			continue
		}

		for _, detail := range workersInfo.Details {
			log.Printf("Worker: %s, Hashrate: %f, Hashrate (1h): %f, Hashrate (24h): %f, Active: %d\n", detail.Worker, detail.Hashrate, detail.Hashrate1h, detail.Hashrate24h, detail.Active)
			hosts, err := database.GetHostsByWorkerID(worker.ID)
			if err != nil {
				log.Printf("Error fetching hosts for account %s: %v", worker.WorkerName, err)
				continue
			}

			matchFound := false
			for _, host := range hosts {
				if detail.Worker == host.WorkerName {
					matchFound = true
					dailyHashInt := detail.Hashrate24h
					log.Printf("Inserting worker hash for worker %s", detail.Worker)
					workerHash := models.WorkerHash{
						FkWorker:   worker.ID,
						FkPoolCoin: poolCoinID,
						DailyHash:  workersInfo.TotalHashrate.Hashrate24h,
						HashDate:   time.Now(),
						FkPool:     pool.ID, // Заполнение fk_pool
					}
					dbSemaphore <- struct{}{}
					go func() {
						defer func() { <-dbSemaphore }()
						err = database.UpdateWorkerHashrate(workerHash, pool.ID)
						if err != nil {
							log.Printf("Error updating worker hashrate for worker %s: %v", detail.Worker, err)
						}
					}()

					log.Printf("Inserting host hash for host %s", detail.Worker)
					hostHash := models.HostHash{
						FkHost:     host.ID,
						DailyHash:  dailyHashInt,
						HashDate:   time.Now(),
						FkPoolCoin: poolCoinID,
						FkPool:     pool.ID, // Заполнение fk_pool
					}
					dbSemaphore <- struct{}{}
					go func() {
						defer func() { <-dbSemaphore }()
						err = database.UpdateHostHashrate(hostHash, pool.ID)
						if err != nil {
							log.Printf("Error updating host hashrate for host %s: %v", detail.Worker, err)
						}
					}()
					break
				}
			}

			if !matchFound {
				log.Printf("WorkerName %s does not match any device of account %s", detail.Worker, worker.WorkerName)
			}
		}

		totalWorkerHash := models.WorkerHash{
			FkWorker:   worker.ID,
			FkPoolCoin: poolCoinID,
			DailyHash:  workersInfo.TotalHashrate.Hashrate24h,
			HashDate:   time.Now(),
			FkPool:     pool.ID, // Заполнение fk_pool
		}
		dbSemaphore <- struct{}{}
		go func() {
			defer func() { <-dbSemaphore }()
			err = database.UpdateWorkerHashrate(totalWorkerHash, pool.ID)
			if err != nil {
				log.Printf("Error updating total hashrate for worker %s: %v", worker.WorkerName, err)
			}
		}()
	}
}
