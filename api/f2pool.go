package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"worker-service/database"
	"worker-service/models"
)

func FetchF2PoolHashrate(baseURL, apiToken, workerName string, currencies []string, workerID string) error {
	for _, currency := range currencies {
		url := fmt.Sprintf("%s/hash_rate/info", baseURL)
		reqBody, _ := json.Marshal(models.HashrateRequest{Currency: currency})
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("F2P-API-SECRET", apiToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

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

		workerHash := models.WorkerHash{
			FkWorker:   workerID,
			FkPoolCoin: currency,
			DailyHash:  hashRateInfo.TotalHashrate.Hashrate24h,
			HashDate:   time.Now(),
		}
		err = database.UpdateWorkerHashrate(workerHash)
		if err != nil {
			return fmt.Errorf("Error updating hashrate for worker %s: %v", workerName, err)
		}
	}
	return nil
}
