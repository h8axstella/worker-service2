package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"worker-service/database"
	"worker-service/models"
)

func FetchHashrate(baseURL, apiKey, accountName string, coins []string, accountID, poolID string) error {
	for _, coin := range coins {
		url := fmt.Sprintf("%s/v1/hashrate/worker?coin=%s", baseURL, coin)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("error creating request for coin %s: %v", coin, err)
		}
		req.Header.Add("X-API-KEY", apiKey)
		response, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching hashrate for coin %s: %v", coin, err)
			continue
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error reading response body for coin %s: %v", coin, err)
			continue
		}

		log.Printf("Response body for coin %s: %s", coin, string(body))
		var hashrateData models.ViaBTCHashrateResponse
		err = json.Unmarshal(body, &hashrateData)
		if err != nil {
			log.Printf("Error unmarshalling response body for coin %s: %v", coin, err)
			continue
		}

		for _, data := range hashrateData.Data.Data {
			log.Printf("Processing worker: %s, WorkerName in data: %s", accountName, data.WorkerName)

			// Check if WorkerName exists in the database
			host, err := database.GetHostByWorkerName(data.WorkerName)
			if err != nil {
				log.Printf("WorkerName %s does not match any device of account %s", data.WorkerName, accountName)
				continue
			}

			log.Printf("Calling GetPoolCoinUUID with poolID: %s and coin: %s", poolID, coin)
			poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
			if err != nil {
				log.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", poolID, coin, err)
				continue
			}
			if poolCoinUUID == "" {
				log.Printf("PoolCoinUUID is empty for pool %s and coin %s", poolID, coin)
				continue
			}

			dailyHashInt, err := strconv.ParseInt(data.Hashrate24Hour, 10, 64)
			if err != nil {
				log.Printf("Error converting hashrate to int64 for worker %s and coin %s: %v", data.WorkerName, coin, err)
				continue
			}

			hostHash := models.HostHash{
				FkHost:     host.ID,
				FkPoolCoin: poolCoinUUID,
				DailyHash:  dailyHashInt,
				HashDate:   time.Now().Format("2006-01-02"),
			}
			log.Printf("Attempting to update host hashrate: %+v", hostHash)
			err = database.UpdateHostHashrate(hostHash)
			if err != nil {
				log.Printf("Error updating hashrate for host %s: %v", data.WorkerName, err)
				continue
			}
			log.Printf("Successfully updated hashrate for host %s", data.WorkerName)
		}
	}
	return nil
}

func FetchAccountHashrateHistory(baseURL, apiKey, coin, startDate, endDate string) ([]models.AccountHashrateHistory, error) {
	var allData []models.AccountHashrateHistory
	page := 1

	for {
		url := fmt.Sprintf("%s/v1/hashrate/history?coin=%s&start_date=%s&end_date=%s&page=%d", strings.TrimRight(baseURL, "/"), coin, startDate, endDate, page)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request for coin %s: %v", coin, err)
		}
		req.Header.Add("X-API-KEY", apiKey)
		response, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching account hashrate history for coin %s: %v", coin, err)
			return nil, err
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error reading response body for coin %s: %v", coin, err)
			return nil, err
		}

		log.Printf("Response body for coin %s: %s", coin, string(body))
		var hashrateHistoryResponse models.AccountHashrateHistoryResponse
		err = json.Unmarshal(body, &hashrateHistoryResponse)
		if err != nil {
			log.Printf("Error unmarshalling response body for coin %s: %v", coin, err)
			return nil, err
		}

		allData = append(allData, hashrateHistoryResponse.Data.Data...)

		if !hashrateHistoryResponse.Data.HasNext {
			break
		}
		page++
	}

	return allData, nil
}

func FetchWorkerHashrateHistory(baseURL, apiKey string, workerID int, coin, startDate, endDate string) ([]models.WorkerHashrateHistory, error) {
	var allData []models.WorkerHashrateHistory
	page := 1

	for {
		url := fmt.Sprintf("%s/v1/hashrate/worker/%d/history?coin=%s&start_date=%s&end_date=%s&page=%d", strings.TrimRight(baseURL, "/"), workerID, coin, startDate, endDate, page)
		log.Printf("Fetching worker hashrate history with URL: %s", url) // Логирование URL
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request for worker %d and coin %s: %v", workerID, coin, err)
		}
		req.Header.Add("X-API-KEY", apiKey)
		response, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching worker hashrate history for worker %d and coin %s: %v", workerID, coin)
			return nil, err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error reading response body for worker %d and coin %s: %v", workerID, coin, err)
			return nil, err
		}

		log.Printf("Response body for worker %d and coin %s: %s", workerID, coin, string(body))
		var hashrateHistoryResponse models.WorkerHashrateHistoryResponse
		err = json.Unmarshal(body, &hashrateHistoryResponse)
		if err != nil {
			log.Printf("Error unmarshalling response body for worker %d and coin %s: %v", workerID, coin, err)
			return nil, err
		}

		for i := range hashrateHistoryResponse.Data.Data {
			hashrateHistoryResponse.Data.Data[i].Hashrate = strings.Split(hashrateHistoryResponse.Data.Data[i].Hashrate, ".")[0]
		}

		allData = append(allData, hashrateHistoryResponse.Data.Data...)

		if !hashrateHistoryResponse.Data.HasNext {
			break
		}
		page++
	}

	return allData, nil
}

func FetchWorkerList(baseURL, apiKey, coin string) ([]models.WorkerListItem, error) {
	var allData []models.WorkerListItem
	page := 1

	for {
		url := fmt.Sprintf("%s/v1/hashrate/worker?coin=%s&page=%d", strings.TrimRight(baseURL, "/"), coin, page)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request for coin %s: %v", coin, err)
		}
		req.Header.Add("X-API-KEY", apiKey)
		response, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching worker list for coin %s: %v", coin, err)
			return nil, err
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error reading response body for coin %s: %v", coin, err)
			return nil, err
		}

		log.Printf("Response body for coin %s: %s", coin, string(body))
		var workerListResponse models.WorkerListResponse
		err = json.Unmarshal(body, &workerListResponse)
		if err != nil {
			log.Printf("Error unmarshalling response body for coin %s: %v", coin, err)
			return nil, err
		}

		allData = append(allData, workerListResponse.Data.Data...)

		if !workerListResponse.Data.HasNext {
			break
		}
		page++
	}

	return allData, nil
}
