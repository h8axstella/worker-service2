package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
	"worker-service/database"
	"worker-service/logger"
	"worker-service/models"
)

func GetF2PoolWorkerHashrate(poolURL, akey, miningUserName, coin string) ([]models.WorkerMiningInfo, error) {
	baseURL, err := url.Parse(poolURL)
	if err != nil {
		return nil, fmt.Errorf("invalid pool URL: %v", err)
	}

	relativePath := path.Join("v2", "hash_rate", "worker", "list")
	baseURL.Path = path.Join(baseURL.Path, relativePath)
	fullURL := baseURL.String()

	logger.InfoLogger.Printf("Requesting URL: %s", fullURL)

	reqBody := map[string]string{
		"mining_user_name": miningUserName,
		"currency":         coin,
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body: %v", err)
	}

	logger.InfoLogger.Printf("Request Body: %s", reqBodyJSON)

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(reqBodyJSON))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("F2P-API-SECRET", akey)

	logger.InfoLogger.Printf("Request Headers: %v", req.Header)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to f2pool: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	logger.InfoLogger.Printf("Response Body: %s", body)

	var response struct {
		Workers []models.WorkerMiningInfo `json:"workers"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	return response.Workers, nil
}

func ProcessF2PoolWorkers(pool models.Pool, worker models.Worker, coin string, workers []models.WorkerMiningInfo, dbSemaphore chan struct{}) error {
	poolCoinUUID, err := database.GetPoolCoinUUIDByFullName(pool.ID, coin)
	if err != nil {
		logger.ErrorLogger.Printf("Error fetching pool coin UUID for pool %s and coin %s: %v", pool.ID, coin, err)
		return err
	}

	var totalHashRate float64

	for _, w := range workers {
		totalHashRate += w.HashRateInfo.H24HashRate

		host, err := database.GetHostByWorkerName(w.HashRateInfo.Name)
		if err != nil {
			logger.WarningLogger.Printf("WorkerName %s does not match any device of account %s", w.HashRateInfo.Name, worker.WorkerName)
			unidentHash := models.UnidentHash{
				HashDate:     time.Now(),
				DailyHash:    w.HashRateInfo.H24HashRate,
				HostWorkerID: worker.ID,
				UnidentName:  w.HashRateInfo.Name,
				FkWorker:     worker.ID,
				FkPoolCoin:   poolCoinUUID,
			}
			dbSemaphore <- struct{}{}
			go func() {
				defer func() { <-dbSemaphore }()
				err = database.InsertUnidentHash(unidentHash)
				if err != nil {
					logger.ErrorLogger.Printf("Error inserting unident hash for worker %s: %v", w.HashRateInfo.Name, err)
				}
			}()
			continue
		}

		hostHash := models.HostHash{
			FkHost:     host.ID,
			FkPoolCoin: poolCoinUUID,
			DailyHash:  w.HashRateInfo.H24HashRate,
			HashDate:   time.Now(),
			FkPool:     pool.ID,
		}
		dbSemaphore <- struct{}{}
		go func() {
			defer func() { <-dbSemaphore }()
			err = database.UpdateHostHashrate(hostHash, pool.ID)
			if err != nil {
				logger.ErrorLogger.Printf("Error updating hashrate for host %s: %v", w.HashRateInfo.Name, err)
			}
		}()
	}

	accountHash := models.WorkerHash{
		FkWorker:   worker.ID,
		FkPoolCoin: poolCoinUUID,
		DailyHash:  totalHashRate,
		HashDate:   time.Now(),
		FkPool:     pool.ID,
	}

	dbSemaphore <- struct{}{}
	go func() {
		defer func() { <-dbSemaphore }()
		err = database.UpdateWorkerHashrate(accountHash, pool.ID)
		if err != nil {
			logger.ErrorLogger.Printf("Error updating account hashrate for worker %s: %v", worker.WorkerName, err)
		}
	}()

	return nil
}
