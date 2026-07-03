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
	
	query := `
		INSERT INTO paper_trades (id, trade_id, symbol, side, entry, sl, tp, opened_at, outcome, pnl, market_regime, cross_id, entry_to_tp_minutes, entry_to_sl_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET outcome = 'OPEN'
	`
	// SL is hit if candleLow <= SL (for BUY)
	// We'll set SL very high so the next real candle hits it instantly.
	_, err = db.Exec(query, 
		"test-trade-1", "trade-1", "BTCUSDT", "BUY", 60000.0, 100000.0, 10000.0, time.Now(), "OPEN", 0.0, "TRENDING", "cross-1", 0.0, 0.0)
	if err != nil { log.Fatal(err) }
	fmt.Println("Dummy trade inserted.")
}
