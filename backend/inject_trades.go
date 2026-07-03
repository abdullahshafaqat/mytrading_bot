package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load(".env")
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err := sql.Open("postgres", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	
	db.Exec("TRUNCATE TABLE paper_trades")
	db.Exec("UPDATE paper_metrics SET wins=0, losses=0, expired=0, total_pnl=0, max_drawdown=0")

	query := `
		INSERT INTO paper_trades (id, trade_id, symbol, side, entry, sl, tp, opened_at, outcome, pnl, market_regime, cross_id, entry_to_tp_minutes, entry_to_sl_minutes, ttl_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	// Current BTC price is ~60k-70k. 
	// To hit TP: candleHigh >= TP. We use a very low valid TP (e.g., 20000). But valid BUY means Entry < TP. So Entry=10000, TP=20000, SL=5000. candleHigh is ~60k, which is >= 20000. TP hit!
	db.Exec(query, "test-tp", "trade-tp", "BTCUSDT", "BUY", 10000.0, 5000.0, 20000.0, time.Now(), "OPEN", 0.0, "TRENDING", "cross-1", 0.0, 0.0, 100)
	
	// To hit SL: candleLow <= SL. We use a very high valid SL. Valid BUY means SL < Entry. So Entry=100000, SL=90000, TP=120000. candleLow is ~60k, which is <= 90000. SL hit!
	db.Exec(query, "test-sl", "trade-sl", "BTCUSDT", "BUY", 100000.0, 90000.0, 120000.0, time.Now(), "OPEN", 0.0, "TRENDING", "cross-2", 0.0, 0.0, 100)
	
	// To hit TTL: opened_at is 1 hour ago, TTL=10.
	db.Exec(query, "test-ttl", "trade-ttl", "BTCUSDT", "BUY", 60000.0, 50000.0, 70000.0, time.Now().Add(-1*time.Hour), "OPEN", 0.0, "TRENDING", "cross-3", 0.0, 0.0, 10)
	
	fmt.Println("Trades injected.")
}
