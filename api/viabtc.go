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

func FetchHashrate(baseURL, apiKey, workerName string, coins []string, workerID string) error {
	for _, coin := range coins {
		url := fmt.Sprintf("%s/v1/hashrate/worker?coin=%s", baseURL, coin)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("Error creating request for coin %s: %v", coin, err)
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

		var hashrateData models.ViaBTCHashrateResponse
		err = json.Unmarshal(body, &hashrateData)
		if err != nil {
			log.Printf("Error unmarshalling response body for coin %s: %v", coin, err)
			continue
		}

		for _, data := range hashrateData.Data.Data {
			if data.WorkerName == workerName {
				hostHash := models.HostHash{
					FkHost:     workerID,
					FkPoolCoin: coin,
					DailyHash:  data.Hashrate24Hour,
					HashDate:   time.Now(),
				}
				err := database.UpdateHostHashrate(hostHash)
				if err != nil {
					return fmt.Errorf("Error updating hashrate for host %s: %v", workerName, err)
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
			return fmt.Errorf("Error creating request for account hashrate for coin %s: %v", coin, err)
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
				Hashrate24Hour float64 `json:"hashrate_24hour,string"`
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

		// Получение poolCoinID
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
			return fmt.Errorf("Error updating account hashrate for worker %s: %v", workerID, err)
		}
	}
	return nil
}
