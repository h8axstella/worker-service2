package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"worker-service/database"
)

func FetchCoins(apiKey string) ([]string, error) {
	url := "https://www.viabtc.com/res/openapi/v1/account"
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
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var res struct {
		Data struct {
			Balance []struct {
				Coin string `json:"coin"`
			} `json:"balance"`
		} `json:"data"`
	}
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

func FetchHashrate(apiKey, workerName string, coins []string, workerID string) error {
	for _, coin := range coins {
		url := fmt.Sprintf("https://www.viabtc.com/res/openapi/v1/hashrate/worker?coin=%s", coin)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request for coin %s: %v\n", coin, err)
			return err
		}
		req.Header.Add("X-API-KEY", apiKey)

		response, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching hashrate for coin %s: %v\n", coin, err)
			continue
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error reading response body for coin %s: %v\n", coin, err)
			continue
		}

		var hashrateData struct {
			Code int `json:"code"`
			Data struct {
				Data []struct {
					Hashrate24Hour float64 `json:"hashrate_24hour,string"`
					WorkerName     string  `json:"worker_name"`
				} `json:"data"`
			} `json:"data"`
		}
		err = json.Unmarshal(body, &hashrateData)
		if err != nil {
			log.Printf("Error unmarshalling response body for coin %s: %v\n", coin, err)
			continue
		}

		for _, data := range hashrateData.Data.Data {
			if data.WorkerName == workerName {
				err := database.UpdateWorkerHashrate(workerID, data.Hashrate24Hour)
				if err != nil {
					log.Printf("Error updating hashrate for worker %s: %v\n", workerName, err)
				}
			}
		}
	}
	return nil
}
