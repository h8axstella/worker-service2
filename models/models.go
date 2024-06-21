package models

import "time"

type Worker struct {
	ID         string  `json:"id"`
	WorkerName string  `json:"worker_name"`
	FkPool     string  `json:"fk_pool"`
	AKey       string  `json:"akey"`
	SKey       *string `json:"skey"`
}

type Pool struct {
	ID       string `json:"id"`
	PoolName string `json:"pool_name"`
	PoolURL  string `json:"pool_url"`
}

type WorkerHash struct {
	FkWorker   string    `json:"fk_worker"`
	DailyHash  int64     `json:"daily_hash"`
	HashDate   time.Time `json:"hash_date"`
	FkPoolCoin string    `json:"fk_pool_coin"`
}

type HostHash struct {
	FkHost     string    `json:"fk_host"`
	DailyHash  int64     `json:"daily_hash"`
	HashDate   time.Time `json:"hash_date"`
	FkPoolCoin string    `json:"fk_pool_coin"`
}

type UnidentHash struct {
	HashDate    time.Time `json:"hash_date"`
	DailyHash   int64     `json:"daily_hash"`
	UnidentName string    `json:"unident_name"`
	FkWorker    string    `json:"fk_worker"`
	FkPoolCoin  string    `json:"fk_pool_coin"`
}

type ViaBTCAccountResponse struct {
	Data struct {
		Balance []struct {
			Coin string `json:"coin"`
		} `json:"balance"`
	} `json:"data"`
}

type ViaBTCHashrateResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TotalPages int `json:"total_page"`
		Data       []struct {
			Hashrate24Hour float64 `json:"hashrate_24hour,string"`
			WorkerName     string  `json:"worker_name"`
		} `json:"data"`
	} `json:"data"`
}

type HashrateRequest struct {
	Currency string `json:"currency"`
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
		Hashrate24h float64 `json:"hashrate_24hour"`
	} `json:"data"`
}

type Host struct {
	ID         string `json:"id"`
	WorkerName string `json:"host_worker"`
}

type WorkersInfo struct {
	TotalCount struct {
		All      int `json:"all"`
		Active   int `json:"active"`
		Inactive int `json:"inactive"`
	} `json:"total_count"`
	TotalHashrate struct {
		Hashrate    float64 `json:"hashrate"`
		Hashrate1h  float64 `json:"hashrate1h"`
		Hashrate24h float64 `json:"hashrate24h"`
	} `json:"total_hashrate"`
	Details []struct {
		Worker      string  `json:"worker"`
		Hashrate    float64 `json:"hashrate"`
		Hashrate1h  float64 `json:"hashrate1h"`
		Hashrate24h float64 `json:"hashrate24h"`
		Active      int     `json:"active"`
	} `json:"details"`
}
