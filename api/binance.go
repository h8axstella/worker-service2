package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"worker-service/models"
)

func generateSignature(data string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func getServerTime(baseURL string) (int64, error) {
	resp, err := http.Get(baseURL + "api/v3/time")
	if err != nil {
		return 0, fmt.Errorf("error getting server time: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	var response struct {
		ServerTime int64 `json:"serverTime"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return response.ServerTime, nil
}

func GetWorkerList(apiKey, apiSecret, algo, userName, baseURL string) ([]models.WorkerMiningInfo, error) {
	serverTime, err := getServerTime(baseURL)
	if err != nil {
		return nil, fmt.Errorf("error getting server time: %w", err)
	}
	timestamp := strconv.FormatInt(serverTime, 10)

	params := map[string]string{
		"algo":      algo,
		"userName":  userName,
		"timestamp": timestamp,
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var data string
	for _, k := range keys {
		data += fmt.Sprintf("%s=%s&", k, params[k])
	}
	data = strings.TrimRight(data, "&")

	signature := generateSignature(data, apiSecret)
	url := fmt.Sprintf("%s/sapi/v1/mining/worker/list?%s&signature=%s", baseURL, data, signature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Binance API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var response models.BinanceWorkersResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("API error: %s", response.Msg)
	}

	var workerInfos []models.WorkerMiningInfo
	for _, worker := range response.Data.WorkerDatas {
		workerInfos = append(workerInfos, models.WorkerMiningInfo{
			HashRateInfo: models.HashRateInfo{
				Name:        worker.WorkerName,
				HashRate:    worker.HashRate,
				H24HashRate: worker.DayHashRate,
			},
			LastShareAt: worker.LastShare,
			Status:      worker.Status,
			Host:        worker.WorkerId,
		})
	}

	return workerInfos, nil
}

func GetWorkerHashrate(apiKey, apiSecret, algo, userName, workerName, baseURL string) (models.WorkerMiningInfo, error) {
	serverTime, err := getServerTime(baseURL)
	if err != nil {
		return models.WorkerMiningInfo{}, fmt.Errorf("error getting server time: %w", err)
	}
	timestamp := strconv.FormatInt(serverTime, 10)

	params := map[string]string{
		"algo":       algo,
		"userName":   userName,
		"workerName": workerName,
		"timestamp":  timestamp,
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var data string
	for _, k := range keys {
		data += fmt.Sprintf("%s=%s&", k, params[k])
	}
	data = strings.TrimRight(data, "&")

	signature := generateSignature(data, apiSecret)
	url := fmt.Sprintf("%s/sapi/v1/mining/worker/detail?%s&signature=%s", baseURL, data, signature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return models.WorkerMiningInfo{}, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.WorkerMiningInfo{}, fmt.Errorf("error sending request to Binance API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		return models.WorkerMiningInfo{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.WorkerMiningInfo{}, fmt.Errorf("error reading response body: %w", err)
	}

	var response models.BinanceWorkerDetailResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return models.WorkerMiningInfo{}, fmt.Errorf("error unmarshalling response: %w", err)
	}

	if response.Code != 0 {
		return models.WorkerMiningInfo{}, fmt.Errorf("API error: %s", response.Msg)
	}

	var workerInfo models.WorkerMiningInfo
	for _, data := range response.Data {
		workerInfo.HashRateInfo.Name = data.WorkerName
		for _, hr := range data.HashrateDatas {
			workerInfo.HashRateInfo.H24HashRate = hr.Hashrate
			workerInfo.HashRateInfo.Reject = hr.Reject
			workerInfo.LastShareAt = hr.Time
		}
	}

	return workerInfo, nil
}
