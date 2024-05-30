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
	client := &http.Client{}
	for _, currency := range currencies {
		url := fmt.Sprintf("%s/hash_rate/info", baseURL)
		reqBody, err := json.Marshal(models.HashrateRequest{Currency: currency})
		if err != nil {
			fmt.Println("Error marshalling request body:", err)
			continue
		}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			fmt.Println("Error creating request:", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("F2P-API-SECRET", apiToken)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			continue
		}

		var hashRateInfo models.F2PoolWorkersInfo
		err = json.Unmarshal(body, &hashRateInfo)
		if err != nil {
			fmt.Println("Error unmarshalling response:", err)
			continue
		}

		if hashRateInfo.TotalHashrate.Hashrate24h == 0 {
			fmt.Printf("API error for %s: no data available\n", currency)
			continue
		}

		dailyHashInt := int64(hashRateInfo.TotalHashrate.Hashrate24h)
		workerHash := models.WorkerHash{
			FkWorker:   workerID,
			FkPoolCoin: currency,
			DailyHash:  dailyHashInt,
			HashDate:   time.Now(),
		}

		err = database.UpdateWorkerHashrate(workerHash)
		if err != nil {
			return fmt.Errorf("error updating hashrate for worker %s: %v", workerName, err)
		}
	}
	return nil
}
