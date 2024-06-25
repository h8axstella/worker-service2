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
		var akey sql.NullString
		var skey sql.NullString
		var fkPool sql.NullString
		err := rows.Scan(&worker.ID, &worker.WorkerName, &akey, &skey, &fkPool)
		if err != nil {
			log.Printf("Error scanning worker row: %v", err)
			return nil, err
		}
		if !akey.Valid || !fkPool.Valid {
			log.Printf("Skipping worker %s due to missing akey or fk_pool", worker.WorkerName)
			continue
		}
		worker.AKey = akey.String
		if skey.Valid {
			worker.SKey = &skey.String
		}
		worker.FkPool = fkPool.String
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

func GetWorkerByName(workerName string) (models.Worker, error) {
	query := `SELECT id, worker_name, akey, skey, fk_pool FROM tb_worker WHERE worker_name = $1`
	var worker models.Worker
	var akey sql.NullString
	var skey sql.NullString
	var fkPool sql.NullString
	err := DB.QueryRow(query, workerName).Scan(&worker.ID, &worker.WorkerName, &akey, &skey, &fkPool)
	if err != nil {
		if err == sql.ErrNoRows {
			return worker, fmt.Errorf("no worker found with WorkerName: %s", workerName)
		}
		return worker, fmt.Errorf("error fetching worker: %v", err)
	}
	if !akey.Valid || !fkPool.Valid {
		return worker, fmt.Errorf("missing akey or fk_pool for worker: %s", workerName)
	}
	worker.AKey = akey.String
	if skey.Valid {
		worker.SKey = &skey.String
	}
	worker.FkPool = fkPool.String
	return worker, nil
}

func UpdateWorkerHashrate(workerHash models.WorkerHash) error {
	query := `
        INSERT INTO tb_worker_hash (fk_worker, daily_hash, hash_date, fk_pool_coin, fk_pool)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (fk_worker, hash_date, fk_pool_coin) DO NOTHING;
    `

	if workerHash.FkPoolCoin == "" {
		log.Printf("FkPoolCoin is empty for worker: %s", workerHash.FkWorker)
		return fmt.Errorf("FkPoolCoin is empty for worker: %s", workerHash.FkWorker)
	}

	log.Printf("Executing query to update worker hashrate: %+v", workerHash)
	result, err := DB.Exec(query, workerHash.FkWorker, workerHash.DailyHash, workerHash.HashDate, workerHash.FkPoolCoin, workerHash.FkPool)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return fmt.Errorf("failed to execute query: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Failed to get rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	log.Printf("Rows affected: %d", rowsAffected)
	return nil
}

func UpdateHostHashrate(hostHash models.HostHash) error {
	query := `
        INSERT INTO tb_host_hash (fk_host, daily_hash, hash_date, fk_pool_coin, fk_pool, host_workerid)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (fk_host, hash_date, fk_pool_coin) DO NOTHING;
    `

	if hostHash.FkPoolCoin == "" {
		log.Printf("FkPoolCoin is empty for host: %s", hostHash.FkHost)
		return fmt.Errorf("FkPoolCoin is empty for host: %s", hostHash.FkHost)
	}

	log.Printf("Executing query to update host hashrate: %+v", hostHash)
	_, err := DB.Exec(query, hostHash.FkHost, hostHash.DailyHash, hostHash.HashDate, hostHash.FkPoolCoin, hostHash.FkPool, hostHash.HostWorkerID)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return fmt.Errorf("failed to execute query: %v", err)
	}

	return nil
}

func InsertUnidentHash(unidentHash models.UnidentHash) error {
	query := `
        INSERT INTO tb_unident_hash (id, hash_date, daily_hash, host_workerid, unident_name, fk_worker, fk_pool_coin, last_edit, status)
        VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW()::timestamp(0), 0)
        ON CONFLICT (hash_date, unident_name, fk_worker, fk_pool_coin) DO NOTHING;
    `

	log.Printf("Executing query to insert unident hashrate: %+v", unidentHash)
	_, err := DB.Exec(query, unidentHash.HashDate, unidentHash.DailyHash, unidentHash.HostWorkerID, unidentHash.UnidentName, unidentHash.FkWorker, unidentHash.FkPoolCoin)
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

func UpdateHostWorkerID(hostWorkerID int, hostID string) error {
	log.Printf("Updating host worker_id for host ID %s with worker ID %d\n", hostID, hostWorkerID)
	tx, err := DB.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	query := `UPDATE tb_host_hash SET host_workerid = $1 WHERE id = $2`
	result, err := tx.Exec(query, hostWorkerID, hostID)
	if err != nil {
		return fmt.Errorf("error updating host worker_id: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error fetching rows affected: %v", err)
	}
	if rowsAffected == 0 {
		log.Printf("No rows were updated for host ID %s\n", hostID)
	} else {
		log.Printf("Successfully updated %d rows for host ID %s\n", rowsAffected, hostID)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
