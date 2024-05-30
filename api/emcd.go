package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"worker-service/models"
)

func GetEmcdWorkersInfo(apiKey, coin, baseURL string) (models.WorkersInfo, error) {
	if apiKey == "" || coin == "" || baseURL == "" {
		return models.WorkersInfo{}, fmt.Errorf("API key, coin, or base URL is empty")
	}

	endpoint, err := url.Parse(fmt.Sprintf("%sv1/%s/workers/%s", baseURL, coin, apiKey))
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("Failed to parse URL: %v", err)
	}
	log.Printf("Requesting URL: %s", endpoint.String())

	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("Failed to create HTTP request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error fetching worker info: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error reading response body: %v", err)
	}

	log.Printf("Response body: %s", string(body))
	var workersInfo models.WorkersInfo
	err = json.Unmarshal(body, &workersInfo)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error unmarshalling response: %v, body: %s", err, string(body))
	}
	return workersInfo, nil
}
