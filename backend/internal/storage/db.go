package storage

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/abdullahshafaqat/trading-bot/internal/logger"
	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func NewDB() (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open failed: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	logger.DB("Connected to PostgreSQL ✅")
	return &DB{conn: conn}, nil
}

func (db *DB) Migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS signals (
		id          VARCHAR(100) PRIMARY KEY,
		trade_id    VARCHAR(100),
		cross_id    VARCHAR(100),
		signal_latency_ms BIGINT,
		market_regime VARCHAR(20),
		entry_to_tp_minutes DECIMAL(20,8),
		entry_to_sl_minutes DECIMAL(20,8),
		symbol      VARCHAR(20)  NOT NULL,
		side        VARCHAR(10)  NOT NULL,
		entry       DECIMAL(20,8) NOT NULL,
		stop_loss   DECIMAL(20,8) NOT NULL,
		take_profit DECIMAL(20,8) NOT NULL,
		reason      TEXT,
		created_at  TIMESTAMP NOT NULL,
		expires_at  TIMESTAMP NOT NULL,
		outcome     VARCHAR(10) DEFAULT 'PENDING'
	);

	ALTER TABLE signals ADD COLUMN IF NOT EXISTS trade_id VARCHAR(100);
	ALTER TABLE signals ADD COLUMN IF NOT EXISTS cross_id VARCHAR(100);
	ALTER TABLE signals ADD COLUMN IF NOT EXISTS signal_latency_ms BIGINT;
	ALTER TABLE signals ADD COLUMN IF NOT EXISTS market_regime VARCHAR(20);
	ALTER TABLE signals ADD COLUMN IF NOT EXISTS entry_to_tp_minutes DECIMAL(20,8);
	ALTER TABLE signals ADD COLUMN IF NOT EXISTS entry_to_sl_minutes DECIMAL(20,8);

	CREATE TABLE IF NOT EXISTS market_data (
		id         SERIAL PRIMARY KEY,
		symbol     VARCHAR(20)   NOT NULL,
		interval   VARCHAR(10)   NOT NULL,
		open       DECIMAL(20,8) NOT NULL,
		high       DECIMAL(20,8) NOT NULL,
		low        DECIMAL(20,8) NOT NULL,
		close      DECIMAL(20,8) NOT NULL,
		volume     DECIMAL(20,8) NOT NULL,
		ts         TIMESTAMP     NOT NULL
	);

	CREATE TABLE IF NOT EXISTS logs (
		id      SERIAL PRIMARY KEY,
		level   VARCHAR(10) NOT NULL,
		message TEXT        NOT NULL,
		ts      TIMESTAMP   NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS paper_trades (
		id                  VARCHAR(100) PRIMARY KEY,
		trade_id            VARCHAR(100),
		symbol              VARCHAR(20)  NOT NULL,
		side                VARCHAR(10)  NOT NULL,
		entry               DECIMAL(20,8) NOT NULL,
		sl                  DECIMAL(20,8) NOT NULL,
		tp                  DECIMAL(20,8) NOT NULL,
		opened_at           TIMESTAMP NOT NULL,
		closed_at           TIMESTAMP,
		outcome             VARCHAR(20),
		pnl                 DECIMAL(20,8),
		market_regime       VARCHAR(20),
		cross_id            VARCHAR(100),
		entry_to_tp_minutes DECIMAL(20,8),
		entry_to_sl_minutes DECIMAL(20,8),
		ttl_minutes         INT DEFAULT 0
	);

	ALTER TABLE paper_trades ADD COLUMN IF NOT EXISTS ttl_minutes INT DEFAULT 0;

	CREATE TABLE IF NOT EXISTS paper_metrics (
		id           SERIAL PRIMARY KEY,
		wins         INT DEFAULT 0,
		losses       INT DEFAULT 0,
		expired      INT DEFAULT 0,
		total_pnl    DECIMAL(20,8) DEFAULT 0,
		max_drawdown DECIMAL(20,8) DEFAULT 0,
		updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
	);

	INSERT INTO paper_metrics (id, wins, losses, expired, total_pnl, max_drawdown, updated_at)
	VALUES (1, 0, 0, 0, 0, 0, NOW())
	ON CONFLICT (id) DO NOTHING;
	`

	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.DB("Tables ready ✅")
	return nil
}

func (db *DB) Close() {
	db.conn.Close()
}
