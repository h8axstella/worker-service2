package models

import "database/sql"

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
	FkWorker   string
	FkPoolCoin string
	DailyHash  int64
	HashDate   string
}

type HostHash struct {
	FkHost     string
	FkPoolCoin string
	DailyHash  int64
	HashDate   string
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
			Hashrate24Hour string `json:"hashrate_24hour"`
			WorkerName     string `json:"worker_name"`
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
	ID           string        `json:"id"`
	WorkerName   string        `json:"host_worker"`
	HostWorkerID sql.NullInt64 `json:"host_workerid"`
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

type AccountHashrateHistory struct {
	Coin       string `json:"coin"`
	Date       string `json:"date"`
	Hashrate   string `json:"hashrate"`
	RejectRate string `json:"reject_rate"`
	PoolCoinID string
}

type AccountHashrateHistoryResponse struct {
	Code    int                        `json:"code"`
	Data    AccountHashrateHistoryData `json:"data"`
	Message string                     `json:"message"`
}

type AccountHashrateHistoryData struct {
	Count     int                      `json:"count"`
	CurrPage  int                      `json:"curr_page"`
	Data      []AccountHashrateHistory `json:"data"`
	HasNext   bool                     `json:"has_next"`
	Total     int                      `json:"total"`
	TotalPage int                      `json:"total_page"`
}

type WorkerHashrateHistory struct {
	Coin       string `json:"coin"`
	Date       string `json:"date"`
	Hashrate   string `json:"hashrate"`
	RejectRate string `json:"reject_rate"`
	PoolCoinID string
}

type WorkerHashrateHistoryResponse struct {
	Code    int                       `json:"code"`
	Data    WorkerHashrateHistoryData `json:"data"`
	Message string                    `json:"message"`
}

type WorkerHashrateHistoryData struct {
	Count     int                     `json:"count"`
	CurrPage  int                     `json:"curr_page"`
	Data      []WorkerHashrateHistory `json:"data"`
	HasNext   bool                    `json:"has_next"`
	Total     int                     `json:"total"`
	TotalPage int                     `json:"total_page"`
}

type WorkerListItem struct {
	WorkerID   int    `json:"worker_id"`
	WorkerName string `json:"worker_name"`
}

type WorkerListResponse struct {
	Code int `json:"code"`
	Data struct {
		Data    []WorkerListItem `json:"data"`
		HasNext bool             `json:"has_next"`
	} `json:"data"`
}
