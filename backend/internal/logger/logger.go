package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Level string

const (
	LevelInfo Level = "INFO"
)

type stdLogger struct {
	file *os.File
}

var (
	mu  sync.Mutex
	std *stdLogger
)

func format(level Level, prefix string, msg string) string {
	return fmt.Sprintf("[%s] [%s] %s", level, prefix, msg)
}

func write(level Level, prefix string, msg string) {
	mu.Lock()
	defer mu.Unlock()

	line := format(level, prefix, msg)

	log.Println(line)

	if std == nil || std.file == nil {
		return
	}

	_, _ = std.file.WriteString(line + "\n")
}

func Init(filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	if filePath == "" {
		filePath = "logs/bot.log"
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("create log dir failed: %w", err)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}

	std = &stdLogger{file: file}
	log.SetFlags(0)
	return nil
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if std == nil || std.file == nil {
		return nil
	}
	err := std.file.Close()
	std = nil
	return err
}

func Bot(v ...any)                   { write(LevelInfo, "BOT", fmt.Sprint(v...)) }
func Botf(format string, v ...any)   { write(LevelInfo, "BOT", fmt.Sprintf(format, v...)) }
func DB(v ...any)                    { write(LevelInfo, "DB", fmt.Sprint(v...)) }
func DBf(format string, v ...any)    { write(LevelInfo, "DB", fmt.Sprintf(format, v...)) }
func WS(v ...any)                    { write(LevelInfo, "WS", fmt.Sprint(v...)) }
func WSf(format string, v ...any)    { write(LevelInfo, "WS", fmt.Sprintf(format, v...)) }
func Signal(v ...any)                { write(LevelInfo, "SIGNAL", fmt.Sprint(v...)) }
func Signalf(format string, v ...any) { write(LevelInfo, "SIGNAL", fmt.Sprintf(format, v...)) }
func Market(v ...any)                { write(LevelInfo, "MARKET", fmt.Sprint(v...)) }
func Marketf(format string, v ...any) { write(LevelInfo, "MARKET", fmt.Sprintf(format, v...)) }
