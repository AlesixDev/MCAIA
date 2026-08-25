package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address        string
	AllowedOrigins []string
	OllamaBaseURL  string
	OllamaModel    string
	Temperature    float64
	NumCtx         int
	Think          bool
	RequestTimeout time.Duration
	MaxUploadBytes int64
	DataDir        string
}

func Load() Config {
	return Config{
		Address:        env("MCAIA_ADDR", "127.0.0.1:8787"),
		AllowedOrigins: []string{env("MCAIA_ORIGIN", "http://localhost:3000")},
		OllamaBaseURL:  env("MCAIA_OLLAMA_URL", "http://127.0.0.1:11434"),
		OllamaModel:    env("MCAIA_OLLAMA_MODEL", "qwen3:14b"),
		Temperature:    envFloat("MCAIA_TEMPERATURE", 0.4),
		NumCtx:         envInt("MCAIA_NUM_CTX", 8192),
		Think:          envBool("MCAIA_OLLAMA_THINK", false),
		RequestTimeout: time.Duration(envInt("MCAIA_TIMEOUT_SECONDS", 180)) * time.Second,
		MaxUploadBytes: int64(envInt("MCAIA_MAX_UPLOAD_MB", 32)) << 20,
		DataDir:        env("MCAIA_DATA_DIR", "data"),
	}
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil {
		return fallback
	}

	return value
}

func envBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(env(key, ""))
	if err != nil {
		return fallback
	}

	return value
}

func envFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(env(key, ""), 64)
	if err != nil {
		return fallback
	}

	return value
}
