package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct{ Server ServerConfig }
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() *Config {
	return &Config{Server: ServerConfig{Port: getEnv("SERVER_PORT", "8080"), ReadTimeout: getDurationEnv("SERVER_READ_TIMEOUT", 15), WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15)}}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" { return v }
	return defaultValue
}

func getDurationEnv(key string, defaultSeconds int) time.Duration {
	if v := os.Getenv(key); v != "" { if s, err := strconv.Atoi(v); err == nil { return time.Duration(s) * time.Second } }
	return time.Duration(defaultSeconds) * time.Second
}
