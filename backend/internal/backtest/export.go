package backtest

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/abdullahshafaqat/trading-bot/internal/strategy"
)

type SignalRow struct {
	Time     time.Time
	Signal   string
	Price    float64
	EMA9     float64
	EMA21    float64
	RSI      float64
	Volume   float64
	VolumeMA float64
	Confirm4h bool
	Reason   string
}

func ExportSignalsCSV(rows []SignalRow, experiment string) (string, error) {
	dir := "backtests"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create backtests dir: %w", err)
	}

	name := "signals.csv"
	if experiment != "" {
		name = fmt.Sprintf("signals_%s.csv", experiment)
	}

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"Time", "Signal", "Price", "EMA9", "EMA21", "RSI",
		"Volume", "VolumeMA", "4H", "Reason",
	}
	if err := w.Write(header); err != nil {
		return "", err
	}

	for _, r := range rows {
		row := []string{
			r.Time.Format("2006-01-02 15:04:05"),
			r.Signal,
			strconv.FormatFloat(r.Price, 'f', 2, 64),
			strconv.FormatFloat(r.EMA9, 'f', 2, 64),
			strconv.FormatFloat(r.EMA21, 'f', 2, 64),
			strconv.FormatFloat(r.RSI, 'f', 2, 64),
			strconv.FormatFloat(r.Volume, 'f', 2, 64),
			strconv.FormatFloat(r.VolumeMA, 'f', 2, 64),
			strconv.FormatBool(r.Confirm4h),
			r.Reason,
		}
		if err := w.Write(row); err != nil {
			return "", err
		}
	}

	return path, nil
}

func rowFromResult(candleTime time.Time, result strategy.Result, confirm4h bool) SignalRow {
	return SignalRow{
		Time:      candleTime,
		Signal:    string(result.Signal),
		Price:     result.Entry,
		EMA9:      result.EMAFast,
		EMA21:     result.EMASlow,
		RSI:       result.RSI,
		Volume:    result.Volume,
		VolumeMA:  result.VolumeMA,
		Confirm4h: confirm4h,
		Reason:    result.Reason,
	}
}
