package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"worker-service/models"

	_ "github.com/lib/pq"
)

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

func GetWorkerKeys(workerID string) (string, *string, error) {
	query := `SELECT akey, skey FROM tb_worker WHERE id = $1`
	var akey string
	var skey sql.NullString
	err := DB.QueryRow(query, workerID).Scan(&akey, &skey)
	if err != nil {
		log.Printf("Error getting worker keys for workerID %s: %v", workerID, err)
		return "", nil, err
	}
	if skey.Valid {
		return akey, &skey.String, nil
	} else {
		return akey, nil, nil
	}
}

func GetActiveWorkers() ([]models.Worker, error) {
	query := `SELECT id, worker_name, akey, skey, fk_pool FROM tb_worker WHERE status = 0`
	rows, err := DB.Query(query)
	if err != nil {
		log.Printf("Error querying active workers: %v", err)
		return nil, err
	}
	defer rows.Close()

	var workers []models.Worker
	for rows.Next() {
		var worker models.Worker
		var skey sql.NullString
		err := rows.Scan(&worker.ID, &worker.WorkerName, &worker.AKey, &skey, &worker.FkPool)
		if err != nil {
			log.Printf("Error scanning worker row: %v", err)
			return nil, err
		}
		if skey.Valid {
			worker.SKey = &skey.String
		}
		workers = append(workers, worker)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating over rows: %v", err)
		return nil, err
	}

	return workers, nil
}

func GetHostByWorkerName(workerName string) (models.Host, error) {
	query := `SELECT id, host_worker FROM tb_host WHERE host_worker = $1`
	var host models.Host
	err := DB.QueryRow(query, workerName).Scan(&host.ID, &host.WorkerName)
	if err != nil {
		if err == sql.ErrNoRows {
			return host, fmt.Errorf("no host found with WorkerName: %s", workerName)
		}
		return host, fmt.Errorf("error fetching host: %v", err)
	}
	return host, nil
}

func GetPoolByID(poolID string) (models.Pool, error) {
	query := `SELECT id, pool_name, pool_url FROM tb_pool WHERE id = $1`
	var pool models.Pool
	err := DB.QueryRow(query, poolID).Scan(&pool.ID, &pool.PoolName, &pool.PoolURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return pool, fmt.Errorf("no pool found with ID: %s", poolID)
		}
		return pool, fmt.Errorf("error fetching pool: %v", err)
	}
	return pool, nil
}

func GetCoinsByPoolID(poolID string) ([]string, error) {
	query := `SELECT c.short_name FROM tb_pool_coin pc JOIN tb_coin c ON c.id = pc.fk_coin WHERE pc.fk_pool = $1`
	rows, err := DB.Query(query, poolID)
	if err != nil {
		log.Printf("Error querying coins for poolID %s: %v", poolID, err)
		return nil, err
	}
	defer rows.Close()

	var coins []string
	for rows.Next() {
		var coin string
		err := rows.Scan(&coin)
		if err != nil {
			log.Printf("Error scanning coin row: %v", err)
			return nil, err
		}
		coins = append(coins, coin)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating over rows: %v", err)
		return nil, err
	}

	return coins, nil
}

func UpdateWorkerHashrate(workerHash models.WorkerHash) error {
	query := `
        INSERT INTO tb_worker_hash (fk_worker, daily_hash, hash_date, fk_pool_coin)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (fk_worker, hash_date, fk_pool_coin) DO UPDATE
        SET daily_hash = EXCLUDED.daily_hash, last_edit = NOW();
    `
	if workerHash.FkPoolCoin == "" {
		log.Printf("FkPoolCoin is empty for worker: %s", workerHash.FkWorker)
		return fmt.Errorf("FkPoolCoin is empty for worker: %s", workerHash.FkWorker)
	}
	_, err := DB.Exec(query, workerHash.FkWorker, workerHash.DailyHash, workerHash.HashDate, workerHash.FkPoolCoin)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return fmt.Errorf("failed to execute query: %v", err)
	}
	return nil
}

func UpdateHostHashrate(hostHash models.HostHash) error {
	query := `
        INSERT INTO tb_host_hash (fk_host, daily_hash, hash_date, fk_pool_coin)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (fk_host, hash_date, fk_pool_coin) DO UPDATE 
        SET daily_hash = EXCLUDED.daily_hash, last_edit = NOW();
    `
	if hostHash.FkPoolCoin == "" {
		log.Printf("FkPoolCoin is empty for host: %s", hostHash.FkHost)
		return fmt.Errorf("FkPoolCoin is empty for host: %s", hostHash.FkHost)
	}
	_, err := DB.Exec(query, hostHash.FkHost, hostHash.DailyHash, hostHash.HashDate, hostHash.FkPoolCoin)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return fmt.Errorf("failed to execute query: %v", err)
	}
	return nil
}

func GetHostsByWorkerID(workerID string) ([]models.Host, error) {
	query := `SELECT id, host_worker FROM tb_host WHERE fk_worker = $1`
	rows, err := DB.Query(query, workerID)
	if err != nil {
		return nil, fmt.Errorf("error fetching hosts: %v", err)
	}
	defer rows.Close()
	var hosts []models.Host
	for rows.Next() {
		var host models.Host
		if err := rows.Scan(&host.ID, &host.WorkerName); err != nil {
			return nil, fmt.Errorf("error scanning host row: %v", err)
		}
		hosts = append(hosts, host)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}
	return hosts, nil
}

func GetPoolCoinUUID(poolID, coin string) (string, error) {
	query := `
        SELECT pc.id
        FROM tb_pool_coin pc
        JOIN tb_coin c ON pc.fk_coin = c.id
        WHERE pc.fk_pool = $1 AND c.short_name = $2
    `
	var poolCoinID string
	err := DB.QueryRow(query, poolID, coin).Scan(&poolCoinID)
	if err != nil {
		log.Printf("Error getting poolCoinID for poolID %s and coin %s: %v", poolID, coin, err)
		return "", err
	}
	return poolCoinID, nil
}
