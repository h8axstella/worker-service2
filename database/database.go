package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type Worker struct {
	ID         string `json:"id"`
	WorkerName string `json:"worker_name"`
	FkPool     string `json:"fk_pool"`
	AKey       string `json:"akey"`
	SKey       string `json:"skey"`
}

type Pool struct {
	ID       string `json:"id"`
	PoolName string `json:"pool_name"`
	PoolURL  string `json:"pool_url"`
}

var DB *sql.DB

func Init() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_SSLMODE"),
	)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Database connected")
}

func GetWorkerKeys(workerID string) (string, string, error) {
	query := `SELECT akey, skey FROM tb_worker WHERE id = $1`
	var akey, skey string
	err := DB.QueryRow(query, workerID).Scan(&akey, &skey)
	if err != nil {
		log.Printf("Error getting worker keys for workerID %s: %v", workerID, err)
		return "", "", err
	}
	return akey, skey, nil
}

func GetActiveWorkers() ([]Worker, error) {
	query := `SELECT id, worker_name, akey, skey, fk_pool FROM tb_worker WHERE status = 0`
	rows, err := DB.Query(query)
	if err != nil {
		log.Printf("Error querying active workers: %v", err)
		return nil, err
	}
	defer rows.Close()

	var workers []Worker
	for rows.Next() {
		var worker Worker
		err := rows.Scan(&worker.ID, &worker.WorkerName, &worker.AKey, &worker.SKey, &worker.FkPool)
		if err != nil {
			log.Printf("Error scanning worker row: %v", err)
			return nil, err
		}
		workers = append(workers, worker)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating over rows: %v", err)
		return nil, err
	}

	return workers, nil
}

func GetPoolByID(poolID string) (Pool, error) {
	query := `SELECT id, pool_name, pool_url FROM tb_pool WHERE id = $1`
	var pool Pool
	err := DB.QueryRow(query, poolID).Scan(&pool.ID, &pool.PoolName, &pool.PoolURL)
	if err != nil {
		log.Printf("Error getting pool by ID %s: %v", poolID, err)
	}
	return pool, err
}

func UpdateWorkerHashrate(workerID string, hashrate float64) error {
	query := `
        INSERT INTO tb_worker_hash (fk_worker, daily_hash, hash_date)
        VALUES ($1, $2, CURRENT_DATE)
        ON CONFLICT (fk_worker, hash_date) DO UPDATE
        SET daily_hash = EXCLUDED.daily_hash, last_edit = NOW();
    `
	_, err := DB.Exec(query, workerID, hashrate)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return fmt.Errorf("failed to execute query: %v", err)
	}
	return nil
}
