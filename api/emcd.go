package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"worker-service/database"
	"worker-service/models"
)

func FetchEMCDHashrate(baseURL, apiKey, workerName, workerID, coin string) error {
	url := fmt.Sprintf("%s/workers/%s", baseURL, apiKey)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	var workersInfo models.EMCDWorkersInfo
	err = json.Unmarshal(body, &workersInfo)
	if err != nil {
		return fmt.Errorf("error unmarshalling response: %v", err)
	}

	for _, detail := range workersInfo.Data {
		if detail.Worker == workerName {
			workerHash := models.WorkerHash{
				FkWorker:   workerID,
				FkPoolCoin: coin,
				DailyHash:  detail.Hashrate24h,
				HashDate:   time.Now(),
			}
			err = database.UpdateWorkerHashrate(workerHash)
			if err != nil {
				return fmt.Errorf("error updating hashrate for worker %s: %v", workerName, err)
			}
		}
	}

	return nil
}
