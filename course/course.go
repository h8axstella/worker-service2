package course

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"time"

	"worker-service/config"
	"worker-service/database"

	_ "github.com/lib/pq"
)

type CoinMarketCapResponse struct {
	Data   map[string]CoinData `json:"data"`
	Status Status              `json:"status"`
}

type CoinData struct {
	Quote Quote `json:"quote"`
}

type Quote struct {
	USD Price `json:"USD"`
}

type Price struct {
	Price float64 `json:"price"`
}

type Status struct {
	Timestamp string `json:"timestamp"`
}

func getBTCPrice(apiKey string) (float64, time.Time, error) {
	url := "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest"
	parameters := "slug=bitcoin&convert=USD"
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", url, parameters), nil)
	if err != nil {
		return 0, time.Time{}, err
	}
	req.Header.Add("Accepts", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", apiKey)
	response, err := client.Do(req)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, time.Time{}, err
	}
	var res CoinMarketCapResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return 0, time.Time{}, err
	}
	data := res.Data["1"]
	price := data.Quote.USD.Price
	timestamp, err := time.Parse(time.RFC3339, res.Status.Timestamp)
	if err != nil {
		return 0, time.Time{}, err
	}
	return price, timestamp, nil
}

func saveBTCPriceToDB(price float64, timestamp time.Time) error {
	ctx := context.Background()
	query := "INSERT INTO tb_btc_rate (rate_date, rate) VALUES ($1, $2)"
	_, err := database.DB.ExecContext(ctx, query, timestamp, price)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // UniqueViolation
				log.Printf("Duplicate entry detected: %v\n", err)
				return nil
			}
		}
		return err
	}
	return nil
}

func ProcessBTCPrice() {
	log.Println("Starting BTC price processing...")
	price, timestamp, err := getBTCPrice(config.AppConfig.BTCKey)
	if err != nil {
		log.Printf("Error getting BTC price: %v\n", err)
		return
	}
	log.Printf("BTC price fetched: %f at %s", price, timestamp)
	err = saveBTCPriceToDB(price, timestamp)
	if err != nil {
		log.Printf("Error saving BTC price to the database: %v\n", err)
		return
	}
	log.Println("BTC price successfully saved to the database.")
}

func ScheduleBTCProcessing() {
	go ProcessBTCPrice()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Println("Scheduled BTC price processing started.")
			ProcessBTCPrice()
		}
	}
}
