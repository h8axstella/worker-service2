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
