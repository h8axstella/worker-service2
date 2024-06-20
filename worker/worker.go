package worker

import (
	"fmt"
	"log"
	"sync"
	"time"
	"worker-service/api"
	"worker-service/common"
	"worker-service/database"
	"worker-service/models"
)

func StartWorkerHashrateProcessor(apiSemaphore, dbSemaphore chan struct{}, maxRetryAttempts int) {
	fmt.Println("Worker hashrate processor started...")
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	log.Printf("Starting initial worker processing at %s\n", time.Now())
	ProcessActiveWorkers(apiSemaphore, dbSemaphore, maxRetryAttempts)
	log.Printf("Initial worker processing completed at %s\n", time.Now())
	for {
		select {
		case <-ticker.C:
			log.Printf("Starting worker processing at %s\n", time.Now())
			ProcessActiveWorkers(apiSemaphore, dbSemaphore, maxRetryAttempts)
			log.Printf("Worker processing completed at %s\n", time.Now())
		}
	}
}

func ProcessActiveWorkers(apiSemaphore, dbSemaphore chan struct{}, maxRetryAttempts int) {
	fmt.Println("Fetching active workers...")

	workers, err := database.GetActiveWorkers()
	if err != nil {
		log.Printf("Error fetching active workers: %v\n", err)
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

	coins, err := database.GetCoinsByPoolID(worker.FkPool)
	if err != nil {
		log.Printf("Error fetching coins for worker %s: %v\n", worker.WorkerName, err)
		return
	}

	switch pool.PoolName {
	case "viabtc", "f2pool":
		processHashrate(pool, worker, coins, akey, dbSemaphore, maxRetryAttempts)
	case "emcd":
		processEmcd(pool, worker, coins, akey, dbSemaphore, maxRetryAttempts)
	}
}

func processHashrate(pool models.Pool, worker models.Worker, coins []string, akey string, dbSemaphore chan struct{}, maxRetryAttempts int) {
	var wg sync.WaitGroup

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()
			err := common.Retry(maxRetryAttempts, 2, func() error {
				return api.FetchWorkerHashrate(pool.PoolURL, akey, worker.WorkerName, []string{coin}, worker.ID, pool.ID)
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

func processEmcd(pool models.Pool, worker models.Worker, coins []string, akey string, dbSemaphore chan struct{}, maxRetryAttempts int) {
	for _, coin := range coins {
		workersInfo, err := api.GetEmcdWorkersInfo(akey, coin, pool.PoolURL)
		if err != nil {
			log.Printf("Error fetching workers info for worker %s and coin %s: %v\n", worker.WorkerName, coin, err)
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
					dailyHashInt := int64(detail.Hashrate24h)
					log.Printf("Inserting worker hash for worker %s", detail.Worker)
					workerHash := models.WorkerHash{
						FkWorker:   worker.ID,
						FkPoolCoin: poolCoinID,
						DailyHash:  int64(workersInfo.TotalHashrate.Hashrate24h),
						HashDate:   time.Now(),
					}
					dbSemaphore <- struct{}{}
					go func() {
						defer func() { <-dbSemaphore }()
						err = database.UpdateWorkerHashrate(workerHash)
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
					}
					dbSemaphore <- struct{}{}
					go func() {
						defer func() { <-dbSemaphore }()
						err = database.UpdateHostHashrate(hostHash)
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
			DailyHash:  int64(workersInfo.TotalHashrate.Hashrate24h),
			HashDate:   time.Now(),
		}
		dbSemaphore <- struct{}{}
		go func() {
			defer func() { <-dbSemaphore }()
			err = database.UpdateWorkerHashrate(totalWorkerHash)
			if err != nil {
				log.Printf("Error updating total hashrate for worker %s: %v", worker.WorkerName, err)
			}
		}()
	}
}
