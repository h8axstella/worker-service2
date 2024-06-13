package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
	"worker-service/common"
	"worker-service/database"
	"worker-service/logger"
	"worker-service/models"
)

func GetEmcdWorkersInfo(apiKey, coin, baseURL string) (models.WorkersInfo, error) {
	if apiKey == "" || coin == "" || baseURL == "" {
		return models.WorkersInfo{}, fmt.Errorf("API key, coin, or base URL is empty")
	}

	endpoint, err := url.Parse(fmt.Sprintf("%sv1/%s/workers/%s", baseURL, coin, apiKey))
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("Failed to parse URL: %v", err)
	}
	logger.InfoLogger.Printf("Requesting URL: %s", endpoint.String())

	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("Failed to create HTTP request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error fetching worker info: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error reading response body: %v", err)
	}

	logger.InfoLogger.Printf("Response body: %s", string(body))
	var workersInfo models.WorkersInfo
	err = json.Unmarshal(body, &workersInfo)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error unmarshalling response: %v, body: %s", err, string(body))
	}
	return workersInfo, nil
}

func FetchEmcdWorkerHashrate(baseURL, apiKey, accountName string, coins []string, accountID, poolID string) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, common.MaxConcurrentRequests)

	for _, coin := range coins {
		wg.Add(1)
		go func(coin string) {
			defer wg.Done()

			totalPages, err := getTotalPagesEmcd(baseURL, apiKey, coin)
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
						return fetchPageDataEmcd(baseURL, apiKey, coin, accountName, accountID, poolID, page)
					})
					if err != nil {
						logger.ErrorLogger.Printf("Failed to fetch page %d for coin %s after %d attempts: %v", page, coin, common.MaxRetryAttempts, err)
					}

					<-semaphore
				}(page)
			}
		}(coin)
	}
	wg.Wait()
	return nil
}

func getTotalPagesEmcd(baseURL, apiKey, coin string) (int, error) {
	apiURL := fmt.Sprintf("%s/v1/%s/workers/%s", baseURL, coin, apiKey)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request for total pages: %v", err)
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error fetching total pages: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body for total pages: %v", err)
	}

	var workersInfo models.WorkersInfo
	err = json.Unmarshal(body, &workersInfo)
	if err != nil {
		logger.ErrorLogger.Printf("Response body: %s", string(body))
		return 0, fmt.Errorf("error unmarshalling response body for total pages: %v", err)
	}

	totalPages := len(workersInfo.Details) / 100 // Assuming 100 workers per page
	if len(workersInfo.Details)%100 != 0 {
		totalPages++
	}
	return totalPages, nil
}

func fetchPageDataEmcd(baseURL, apiKey, coin, accountName, accountID, poolID string, page int) error {
	apiURL := fmt.Sprintf("%s/v1/%s/workers/%s?page=%d", baseURL, coin, apiKey, page)
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("error creating request for coin %s: %v", coin, err)
	}

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching hashrate for coin %s: %v", coin, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body for coin %s: %v", coin, err)
	}

	var workersInfo models.WorkersInfo
	err = json.Unmarshal(body, &workersInfo)
	if err != nil {
		return fmt.Errorf("error unmarshalling response body for coin %s: %v", coin, err)
	}

	for _, data := range workersInfo.Details {
		logger.InfoLogger.Printf("Processing worker: %s, WorkerName in data: %s", accountName, data.Worker)

		host, err := database.GetHostByWorkerName(data.Worker)
		if err != nil {
			logger.WarningLogger.Printf("WorkerName %s does not match any device of account %s", data.Worker, accountName)
			unidentHash := models.UnidentHash{
				HashDate:    time.Now(),
				DailyHash:   int64(data.Hashrate24h),
				UnidentName: data.Worker,
				FkWorker:    accountID,
				FkPoolCoin:  poolID,
			}
			err = database.InsertUnidentHash(unidentHash)
			if err != nil {
				logger.ErrorLogger.Printf("Error inserting unident hash for worker %s: %v", data.Worker, err)
			}
			continue
		}

		poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
		if err != nil {
			logger.ErrorLogger.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", poolID, coin, err)
			continue
		}

		dailyHashInt := int64(data.Hashrate24h)
		hostHash := models.HostHash{
			FkHost:     host.ID,
			FkPoolCoin: poolCoinUUID,
			DailyHash:  dailyHashInt,
			HashDate:   time.Now(),
		}
		logger.InfoLogger.Printf("Attempting to update host hashrate: %+v", hostHash)
		err = database.UpdateHostHashrate(hostHash)
		if err != nil {
			logger.ErrorLogger.Printf("Error updating hashrate for host %s: %v", data.Worker, err)
			continue
		}
		logger.InfoLogger.Printf("Successfully updated hashrate for host %s", data.Worker)
	}

	return nil
}
