package models

import "time"

type Worker struct {
	ID         string `json:"id"`
	WorkerName string `json:"worker_name"`
	FkPool     string `json:"fk_pool"`
	AKey       string `json:"akey"`
	SKey       string `json:"skey"`
}

type Pool struct {
	ID       string    `json:"id"`
	PoolName string    `json:"pool_name"`
	PoolURL  string    `json:"pool_url"`
	LastEdit time.Time `json:"last_edit"`
	Status   int       `json:"status"`
}

type WorkerHash struct {
	ID        string    `json:"id"`
	FkWorker  string    `json:"fk_worker"`
	Coin      string    `json:"coin"`
	DailyHash float64   `json:"daily_hash"`
	LastEdit  time.Time `json:"last_edit"`
}

type ViaBTCAccountResponse struct {
	Data struct {
		Balance []struct {
			Coin string `json:"coin"`
		} `json:"balance"`
	} `json:"data"`
}

type ViaBTCHashrateResponse struct {
	Code int `json:"code"`
	Data struct {
		Data []struct {
			Hashrate24Hour float64 `json:"hashrate_24hour,string"`
			WorkerName     string  `json:"worker_name"`
		} `json:"data"`
	} `json:"data"`
}

type HashrateRequest struct {
	Currency   string `json:"currency"`
	UserName   string `json:"user_name"`
	WorkerName string `json:"worker_name"`
}

type F2PoolWorkersInfo struct {
	TotalCount struct {
		All      int `json:"all"`
		Active   int `json:"active"`
		Inactive int `json:"inactive"`
	} `json:"total_count"`
	TotalHashrate struct {
		Hashrate24h float64 `json:"hashrate24h"`
	} `json:"total_hashrate"`
	Details []struct {
		Worker      string  `json:"worker"`
		Hashrate24h float64 `json:"hashrate24h"`
		Active      int     `json:"active"`
	} `json:"details"`
}

type EMCDWorkersInfo struct {
	Data []struct {
		Worker      string  `json:"worker"`
		Hashrate24h float64 `json:"hashrate24h"`
	} `json:"data"`
}
