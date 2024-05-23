package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"worker-service/database"
	"worker-service/models"
)

func FetchF2PoolHashrate(baseURL, apiToken, workerName string, currencies []string, workerID string) error {
	for _, currency := range currencies {
		url := fmt.Sprintf("%s/hash_rate/info", baseURL)
		reqBody, err := json.Marshal(models.HashrateRequest{Currency: currency, WorkerName: workerName})
		if err != nil {
			return fmt.Errorf("error marshalling request body: %v", err)
		}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("F2P-API-SECRET", apiToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error sending request: %v", err)
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				err = cerr
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}

		var hashRateInfo models.F2PoolWorkersInfo
		err = json.Unmarshal(body, &hashRateInfo)
		if err != nil {
			return fmt.Errorf("error unmarshalling response: %v", err)
		}

		if hashRateInfo.TotalHashrate.Hashrate24h == 0 {
			fmt.Printf("API error for %s: no data available\n", currency)
			continue
		}

		workerHash := models.WorkerHash{
			FkWorker:  workerID,
			Coin:      currency,
			DailyHash: hashRateInfo.TotalHashrate.Hashrate24h,
			LastEdit:  time.Now(),
		}

		err = database.UpdateWorkerHashrate(workerHash)
		if err != nil {
			return fmt.Errorf("Error updating hashrate for worker %s: %v", workerName, err)
		}
	}
	return nil
}
