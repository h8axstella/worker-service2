package models

import "database/sql"

type Worker struct {
	ID         string  `json:"id"`
	WorkerName string  `json:"worker_name"`
	AKey       string  `json:"akey"`
	SKey       *string `json:"skey,omitempty"`
	FkPool     string  `json:"fk_pool"`
}

type Host struct {
	ID           string        `json:"id"`
	WorkerName   string        `json:"worker_name"`
	FkPool       string        `json:"fk_pool"`
	HostWorkerID sql.NullInt64 `json:"host_workerid"`
}

type WorkerHash struct {
	FkWorker   string  `json:"fk_worker"`
	FkPoolCoin string  `json:"fk_pool_coin"`
	DailyHash  float64 `json:"daily_hash"`
	HashDate   string  `json:"hash_date"`
	FkPool     string  `json:"fk_pool"`
}

type HostHash struct {
	FkHost       string  `json:"fk_host"`
	FkPoolCoin   string  `json:"fk_pool_coin"`
	DailyHash    float64 `json:"daily_hash"`
	HashDate     string  `json:"hash_date"`
	FkPool       string  `json:"fk_pool"`
	HostWorkerID string  `json:"host_worker_id"`
}

type AccountHashrateHistory struct {
	Date     string `json:"date"`
	Hashrate string `json:"hashrate"`
}

type WorkerHashrateHistory struct {
	Date     string `json:"date"`
	Hashrate string `json:"hashrate"`
}

type AccountHashrateHistoryResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TotalPage int                      `json:"total_page"`
		Total     int                      `json:"total"`
		HasNext   bool                     `json:"has_next"`
		CurrPage  int                      `json:"curr_page"`
		Count     int                      `json:"count"`
		Data      []AccountHashrateHistory `json:"data"`
	} `json:"data"`
}

type WorkerHashrateHistoryResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TotalPage int                     `json:"total_page"`
		Total     int                     `json:"total"`
		HasNext   bool                    `json:"has_next"`
		CurrPage  int                     `json:"curr_page"`
		Count     int                     `json:"count"`
		Data      []WorkerHashrateHistory `json:"data"`
	} `json:"data"`
}

type WorkerListItem struct {
	WorkerID   int    `json:"worker_id"`
	WorkerName string `json:"worker_name"`
}

type WorkerListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TotalPage int              `json:"total_page"`
		Total     int              `json:"total"`
		HasNext   bool             `json:"has_next"`
		CurrPage  int              `json:"curr_page"`
		Count     int              `json:"count"`
		Data      []WorkerListItem `json:"data"`
	} `json:"data"`
}

type Pool struct {
	ID       string `json:"id"`
	PoolName string `json:"pool_name"`
	PoolURL  string `json:"pool_url"`
}

type ViaBTCHashrateResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TotalPage int  `json:"total_page"`
		Total     int  `json:"total"`
		HasNext   bool `json:"has_next"`
		CurrPage  int  `json:"curr_page"`
		Count     int  `json:"count"`
		Data      []struct {
			WorkerID       int    `json:"worker_id"`
			WorkerName     string `json:"worker_name"`
			Hashrate24Hour string `json:"hashrate_24h"`
		} `json:"data"`
	} `json:"data"`
}

type WorkersInfo struct {
	WorkerID    string `json:"worker_id"`
	WorkerName  string `json:"worker_name"`
	WorkerGroup string `json:"worker_group"`
	WorkerType  string `json:"worker_type"`
	Status      string `json:"status"`
	LastShare   int64  `json:"last_share"`
}

type UnidentHash struct {
	HashDate     string `json:"hash_date"`
	DailyHash    int64  `json:"daily_hash"`
	HostWorkerID string `json:"host_worker_id"`
	UnidentName  string `json:"unident_name"`
	FkWorker     string `json:"fk_worker"`
	FkPoolCoin   string `json:"fk_pool_coin"`
}
