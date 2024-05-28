package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"worker-service/models"
)

const emcdBaseURL = "https://api.emcd.io/v1"

var emcdCoins = []string{"btc", "bch", "ltc", "dash", "etc", "doge", "kas"}

func GetEmcdWorkersInfo(apiKey, coin string) (models.WorkersInfo, error) {
	url := fmt.Sprintf("%s/%s/workers/%s", emcdBaseURL, coin, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return models.WorkersInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.WorkersInfo{}, fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.WorkersInfo{}, err
	}

	var workersInfo models.WorkersInfo
	err = json.Unmarshal(body, &workersInfo)
	if err != nil {
		return models.WorkersInfo{}, fmt.Errorf("error unmarshalling response: %v, body: %s", err, string(body))
	}

	return workersInfo, nil
}
