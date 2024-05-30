package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

		hosts, err := database.GetHostsByWorkerID(accountID)
		if err != nil {
			log.Printf("Error fetching hosts for account %s: %v", accountName, err)
			continue
		}

		for _, data := range hashrateData.Data.Data {
			log.Printf("Processing worker: %s, WorkerName in data: %s", accountName, data.WorkerName)
			for _, host := range hosts {
				if data.WorkerName == host.WorkerName {
					poolCoinUUID, err := database.GetPoolCoinUUID(poolID, coin)
					if err != nil {
						log.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", poolID, coin, err)
						continue
					}
					dailyHashInt := int64(data.Hashrate24Hour)
					hostHash := models.HostHash{
						FkHost:     host.ID,
						FkPoolCoin: poolCoinUUID,
						DailyHash:  dailyHashInt,
						HashDate:   time.Now(),
					}
					log.Printf("Attempting to update host hashrate: %+v", hostHash)
					err = database.UpdateHostHashrate(hostHash)
					if err != nil {
						log.Printf("Error updating hashrate for host %s: %v", data.WorkerName, err)
						continue
					}
					log.Printf("Successfully updated hashrate for host %s", data.WorkerName)
				} else {
					log.Printf("WorkerName %s does not match any device of account %s", data.WorkerName, accountName)
				}
			}
		}
	}
	return nil
}

func FetchAccountHashrate(baseURL, apiKey string, coins []string, workerID, poolID string) error {
	for _, coin := range coins {
		url := fmt.Sprintf("%s/v1/hashrate?coin=%s", baseURL, coin)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("error creating request for account hashrate for coin %s: %v", coin, err)
		}
		req.Header.Add("X-API-KEY", apiKey)
		response, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching account hashrate for coin %s: %v", coin, err)
			continue
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error reading response body for account hashrate for coin %s: %v", coin, err)
			continue
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
			log.Printf("Error unmarshalling response body for account hashrate for coin %s: %v", coin, err)
			continue
		}
		if accountHashrateData.Data.Hashrate24Hour == 0 {
			log.Printf("API error for account hashrate for coin %s: no data available\n", coin)
			continue
		}
		poolCoinID, err := database.GetPoolCoinID(poolID, coin)
		if err != nil {
			log.Printf("Coin %s does not exist in tb_pool_coin\n", coin)
			continue
		}
		workerHash := models.WorkerHash{
			FkWorker:   workerID,
			FkPoolCoin: poolCoinID,
			DailyHash:  accountHashrateData.Data.Hashrate24Hour,
			HashDate:   time.Now(),
		}
		log.Printf("Updating worker hashrate with: %+v\n", workerHash)
		err = database.UpdateWorkerHashrate(workerHash)
		if err != nil {
			return fmt.Errorf("error updating account hashrate for worker %s: %v", workerID, err)
		}
	}
	return nil
}
