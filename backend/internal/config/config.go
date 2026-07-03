package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv string

	Exchange string
	Symbol   string

	PrimaryTimeframe string
	ConfirmTimeframe string

	RSIPeriod int

	EMAFast int
	EMASlow int

	ATRPeriod int

	VolumePeriod int

	SignalTTLHours int

	SignalCooldownMinutes int

	CrossWindow int

	RiskPerTrade float64

	DailyLossLimit float64

	TelegramEnabled bool

	TelegramBotToken string
	TelegramChatIDs  string
}

func Load() *Config {

	err := godotenv.Load()

	if err != nil {
		log.Println("env file not found")
	}

	return &Config{
		AppEnv: getString("APP_ENV"),

		Exchange: getString("EXCHANGE"),

		Symbol: getString("SYMBOL"),

		PrimaryTimeframe: getString("PRIMARY_TIMEFRAME"),

		ConfirmTimeframe: getString("CONFIRM_TIMEFRAME"),

		RSIPeriod: getInt("RSI_PERIOD"),

		EMAFast: getInt("EMA_FAST"),

		EMASlow: getInt("EMA_SLOW"),

		ATRPeriod: getInt("ATR_PERIOD"),

		VolumePeriod: getInt("VOLUME_PERIOD"),

		SignalTTLHours: getInt("SIGNAL_TTL_HOURS"),

		SignalCooldownMinutes: getInt("SIGNAL_COOLDOWN_MINUTES"),

		CrossWindow: getInt("CROSS_WINDOW"),

		RiskPerTrade: getFloat("RISK_PER_TRADE"),

		DailyLossLimit: getFloat("DAILY_LOSS_LIMIT"),

		TelegramEnabled: getBool("TELEGRAM_ENABLED"),

		TelegramBotToken: getString("TELEGRAM_BOT_TOKEN"),

		TelegramChatIDs: getString("TELEGRAM_CHAT_IDS"),
	}
}

func getString(key string) string {
	return os.Getenv(key)
}

func getInt(key string) int {
	v, _ := strconv.Atoi(os.Getenv(key))
	return v
}

func getFloat(key string) float64 {
	v, _ := strconv.ParseFloat(os.Getenv(key), 64)
	return v
}

func getBool(key string) bool {
	v, _ := strconv.ParseBool(os.Getenv(key))
	return v
}
