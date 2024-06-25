package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
	"worker-service/common"
	"worker-service/database"
	"worker-service/logger"
	"worker-service/models"
)

func FetchWorkerHashrate(baseURL, apiKey, accountName string, coins []string, accountID, poolID string) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, common.MaxConcurrentRequests)

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()

			totalPages, err := getTotalPages(baseURL, apiKey, coin)
			if err != nil {
				logger.ErrorLogger.Printf("Error getting total pages for coin %s: %v", coin, err)
				return
			}

			for page := 1; page <= totalPages; page++ {
				wg.Add(1)
				go func(page int) {
					defer wg.Done()
					semaphore <- struct{}{}

					err := common.Retry(common.MaxRetryAttempts, 2, func() error {
						return fetchPageData(baseURL, apiKey, coin, accountName, accountID, poolID, page)
					})
					if err != nil {
						logger.ErrorLogger.Printf("Failed to fetch page %d for coin %s after %d attempts: %v", page, coin, common.MaxRetryAttempts, err)
					} else {
						logger.InfoLogger.Printf("Successfully fetched page %d for coin %s", page, coin)
					}

					<-semaphore
				}(page)
			}
		}(coin)
	}
	wg.Wait()
	return nil
}

func fetchPageData(baseURL, apiKey, coin, accountName, accountID, poolID string, page int) error {
	url := fmt.Sprintf("%s/v1/hashrate/worker?coin=%s&page=%d", baseURL, coin, page)
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:        common.MaxConcurrentRequests,
			MaxIdleConnsPerHost: common.MaxConcurrentRequests,
		},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for coin %s: %v", coin, err)
	}
	req.Header.Add("X-API-KEY", apiKey)
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching hashrate for coin %s: %v", coin, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body for coin %s: %v", coin, err)
	}

	logger.InfoLogger.Printf("Response body for coin %s (page %d): %s", coin, page, string(body))
	var hashrateData models.ViaBTCHashrateResponse
	err = json.Unmarshal(body, &hashrateData)
	if err != nil {
		return fmt.Errorf("error unmarshalling response body for coin %s: %v", coin, err)
	}

	for _, data := range hashrateData.Data.Data {
		logger.InfoLogger.Printf("Processing worker: %s, WorkerName in data: %s", accountName, data.WorkerName)

		host, err := database.GetHostByWorkerName(data.WorkerName)
		if err != nil {
			logger.WarningLogger.Printf("WorkerName %s does not match any device of account %s", data.WorkerName, accountName)
			poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
			if err != nil {
				logger.ErrorLogger.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", poolID, coin, err)
				continue
			}
			unidentHash := models.UnidentHash{
				HashDate:     time.Now(),
				DailyHash:    data.Hashrate24Hour,
				HostWorkerID: accountID, // Заменяем worker.ID на accountID
				UnidentName:  data.WorkerName,
				FkWorker:     accountID,
				FkPoolCoin:   poolCoinUUID,
			}
			err = database.InsertUnidentHash(unidentHash)
			if err != nil {
				logger.ErrorLogger.Printf("Error inserting unident hash for worker %s: %v", data.WorkerName, err)
			}
			continue
		}

		poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
		if err != nil {
			logger.ErrorLogger.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", poolID, coin, err)
			continue
		}

		dailyHashFloat := data.Hashrate24Hour // Изменено на float64
		hostHash := models.HostHash{
			FkHost:     host.ID,
			FkPoolCoin: poolCoinUUID,
			DailyHash:  dailyHashFloat,
			HashDate:   time.Now(),
			FkPool:     poolID,
		}
		logger.InfoLogger.Printf("Attempting to update host hashrate: %+v", hostHash)
		err = database.UpdateHostHashrate(hostHash, poolID)
		if err != nil {
			logger.ErrorLogger.Printf("Error updating hashrate for host %s: %v", data.WorkerName, err)
			continue
		}
		logger.InfoLogger.Printf("Successfully updated hashrate for host %s", data.WorkerName)
	}

	return nil
}

func getTotalPages(baseURL, apiKey, coin string) (int, error) {
	url := fmt.Sprintf("%s/v1/hashrate/worker?coin=%s&page=1", baseURL, coin)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request for total pages: %v", err)
	}
	req.Header.Add("X-API-KEY", apiKey)
	client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			MaxIdleConns:        common.MaxConcurrentRequests,
			MaxIdleConnsPerHost: common.MaxConcurrentRequests,
		},
	}
	response, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error fetching total pages: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body for total pages: %v", err)
	}

	var hashrateData models.ViaBTCHashrateResponse
	err = json.Unmarshal(body, &hashrateData)
	if err != nil {
		logger.ErrorLogger.Printf("Response body: %s", string(body))
		return 0, fmt.Errorf("error unmarshalling response body for total pages: %v", err)
	}

	totalPages := hashrateData.Data.TotalPages
	return totalPages, nil
}

func FetchOverallAccountHashrate(baseURL, apiKey string, coins []string, workerID, poolID string) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, common.MaxConcurrentRequests)

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()
			semaphore <- struct{}{}
			url := fmt.Sprintf("%s/v1/hashrate?coin=%s", baseURL, coin)
			client := &http.Client{
				Timeout: time.Second * 30,
				Transport: &http.Transport{
					MaxIdleConns:        common.MaxConcurrentRequests,
					MaxIdleConnsPerHost: common.MaxConcurrentRequests,
				},
			}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				logger.ErrorLogger.Printf("Error creating request for account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}
			req.Header.Add("X-API-KEY", apiKey)
			response, err := client.Do(req)
			if err != nil {
				logger.ErrorLogger.Printf("Error fetching account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				logger.ErrorLogger.Printf("Error reading response body for account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}

			var accountHashrateData struct {
				Code int `json:"code"`
				Data struct {
					Hashrate24Hour float64 `json:"hashrate_24hour,string"` // Изменено на float64
				} `json:"data"`
				Message string `json:"message"`
			}
			err = json.Unmarshal(body, &accountHashrateData)
			if err != nil {
				logger.ErrorLogger.Printf("Response body: %s", string(body))
				logger.ErrorLogger.Printf("Error unmarshalling response body for account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}
			if accountHashrateData.Data.Hashrate24Hour == 0 {
				logger.ErrorLogger.Printf("API error for account hashrate for coin %s: no data available\n", coin)
				<-semaphore
				return
			}
			poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
			if err != nil {
				logger.ErrorLogger.Printf("Coin %s does not exist in tb_pool_coin\n", coin)
				<-semaphore
				return
			}
			workerHash := models.WorkerHash{
				FkWorker:   workerID,
				FkPoolCoin: poolCoinUUID,
				DailyHash:  accountHashrateData.Data.Hashrate24Hour, // Используется float64
				HashDate:   time.Now(),
				FkPool:     poolID,
			}
			logger.InfoLogger.Printf("Updating worker hashrate with: %+v\n", workerHash)
			err = database.UpdateWorkerHashrate(workerHash, poolID)
			if err != nil {
				logger.ErrorLogger.Printf("Error updating account hashrate for worker %s: %v", workerID, err)
				<-semaphore
				return
			}
			<-semaphore
		}(coin)
	}
	wg.Wait()
	return nil
}
