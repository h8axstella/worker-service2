package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"worker-service/models"
)

const emcdBaseURL = "https://api.emcd.io/v1"

func GetEmcdWorkersInfo(apiKey, coin string) (models.WorkersInfo, error) {
	url := fmt.Sprintf("%s/%s/workers/%s", emcdBaseURL, coin, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error fetching worker info: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error reading response body: %v", err)
	}
	var workersInfo models.WorkersInfo
	err = json.Unmarshal(body, &workersInfo)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error unmarshalling response: %v, body: %s", err, string(body))
	}
	return workersInfo, nil
}
