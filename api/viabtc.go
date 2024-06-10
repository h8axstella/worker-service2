package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"worker-service/database"
	"worker-service/models"
)

const (
	maxConcurrentRequests = 500
	maxRetryAttempts      = 3
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        maxConcurrentRequests,
			MaxIdleConnsPerHost: maxConcurrentRequests,
		},
	}

	infoLogger    = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	warningLogger = log.New(os.Stderr, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger   = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func FetchWorkerHashrate(baseURL, apiKey, accountName string, coins []string, accountID, poolID string) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentRequests)

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()

			totalPages, err := getTotalPages(baseURL, apiKey, coin)
			if err != nil {
				errorLogger.Printf("Error getting total pages for coin %s: %v", coin, err)
				return
			}

			for page := 1; page <= totalPages; page++ {
				wg.Add(1)
				go func(page int) {
					defer wg.Done()
					semaphore <- struct{}{}

					for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
						err := fetchPageData(baseURL, apiKey, coin, accountName, poolID, page)
						if err == nil {
							break
						}
						if attempt == maxRetryAttempts {
							errorLogger.Printf("Failed to fetch page %d for coin %s after %d attempts: %v", page, coin, maxRetryAttempts, err)
						} else {
							warningLogger.Printf("Retry attempt %d for page %d of coin %s", attempt, page, coin)
						}
					}

					<-semaphore
				}(page)
			}
		}(coin)
	}
	wg.Wait()
	return nil
}

func fetchPageData(baseURL, apiKey, coin, accountName, poolID string, page int) error {
	url := fmt.Sprintf("%s/v1/hashrate/worker?coin=%s&page=%d", baseURL, coin, page)
	client := httpClient
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

	infoLogger.Printf("Response body for coin %s: %s", coin, string(body))
	var hashrateData models.ViaBTCHashrateResponse
	err = json.Unmarshal(body, &hashrateData)
	if err != nil {
		return fmt.Errorf("error unmarshalling response body for coin %s: %v", coin, err)
	}

	for _, data := range hashrateData.Data.Data {
		infoLogger.Printf("Processing worker: %s, WorkerName in data: %s", accountName, data.WorkerName)

		host, err := database.GetHostByWorkerName(data.WorkerName)
		if err != nil {
			warningLogger.Printf("WorkerName %s does not match any device of account %s", data.WorkerName, accountName)
			continue
		}

		poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
		if err != nil {
			errorLogger.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", poolID, coin, err)
			continue
		}

		dailyHashInt := int64(data.Hashrate24Hour)
		hostHash := models.HostHash{
			FkHost:     host.ID,
			FkPoolCoin: poolCoinUUID,
			DailyHash:  dailyHashInt,
			HashDate:   time.Now(),
		}
		infoLogger.Printf("Attempting to update host hashrate: %+v", hostHash)
		err = database.UpdateHostHashrate(hostHash)
		if err != nil {
			errorLogger.Printf("Error updating hashrate for host %s: %v", data.WorkerName, err)
			continue
		}
		infoLogger.Printf("Successfully updated hashrate for host %s", data.WorkerName)
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
	response, err := httpClient.Do(req)
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
		errorLogger.Printf("Response body: %s", string(body))
		return 0, fmt.Errorf("error unmarshalling response body for total pages: %v", err)
	}

	totalPages := hashrateData.Data.TotalPages
	return totalPages, nil
}

func FetchOverallAccountHashrate(baseURL, apiKey string, coins []string, workerID, poolID string) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrentRequests)

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()
			semaphore <- struct{}{}
			url := fmt.Sprintf("%s/v1/hashrate?coin=%s", baseURL, coin)
			client := httpClient
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				errorLogger.Printf("Error creating request for account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}
			req.Header.Add("X-API-KEY", apiKey)
			response, err := client.Do(req)
			if err != nil {
				errorLogger.Printf("Error fetching account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				errorLogger.Printf("Error reading response body for account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}

			var accountHashrateData struct {
				Code int `json:"code"`
				Data struct {
					Hashrate24Hour int64 `json:"hashrate_24hour,string"`
				} `json:"data"`
				Message string `json:"message"`
			}
			err = json.Unmarshal(body, &accountHashrateData)
			if err != nil {
				errorLogger.Printf("Response body: %s", string(body))
				errorLogger.Printf("Error unmarshalling response body for account hashrate for coin %s: %v", coin, err)
				<-semaphore
				return
			}
			if accountHashrateData.Data.Hashrate24Hour == 0 {
				errorLogger.Printf("API error for account hashrate for coin %s: no data available\n", coin)
				<-semaphore
				return
			}
			poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
			if err != nil {
				errorLogger.Printf("Coin %s does not exist in tb_pool_coin\n", coin)
				<-semaphore
				return
			}
			workerHash := models.WorkerHash{
				FkWorker:   workerID,
				FkPoolCoin: poolCoinUUID,
				DailyHash:  accountHashrateData.Data.Hashrate24Hour,
				HashDate:   time.Now(),
			}
			infoLogger.Printf("Updating worker hashrate with: %+v\n", workerHash)
			err = database.UpdateWorkerHashrate(workerHash)
			if err != nil {
				errorLogger.Printf("Error updating account hashrate for worker %s: %v", workerID, err)
				<-semaphore
				return
			}
			<-semaphore
		}(coin)
	}
	wg.Wait()
	return nil
}
