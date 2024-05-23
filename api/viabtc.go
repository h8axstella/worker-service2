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

func FetchCoins(baseURL, apiKey string) ([]string, error) {
	url := fmt.Sprintf("%s/v1/account", baseURL)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Add("X-API-KEY", apiKey)
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer func() {
		if cerr := response.Body.Close(); cerr != nil {
			err = cerr
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var res models.ViaBTCAccountResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %v", err)
	}

	var coins []string
	for _, b := range res.Data.Balance {
		coins = append(coins, b.Coin)
	}
	return coins, nil
}

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
		defer func() {
			if cerr := response.Body.Close(); cerr != nil {
				err = cerr
			}
		}()

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
				workerHash := models.WorkerHash{
					FkWorker:  workerID,
					Coin:      coin,
					DailyHash: data.Hashrate24Hour,
					LastEdit:  time.Now(),
				}
				err := database.UpdateWorkerHashrate(workerHash)
				if err != nil {
					return fmt.Errorf("Error updating hashrate for worker %s: %v", workerName, err)
				}
			}
		}
	}
	return nil
}
