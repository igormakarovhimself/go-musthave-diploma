package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RunAddress            string
	DatabaseURI           string
	AccrualSystemAddress  string
	JWTSecret             string
	AccrualWorkers        int
	AccrualMaxConcurrency int
	AccrualCheckInterval  time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "PostgreSQL connection URI")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "Accrual system address")
	flag.StringVar(&cfg.JWTSecret, "j", "", "JWT secret key")

	flag.Parse()

	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		cfg.RunAddress = envAddr
	}

	if envDB := os.Getenv("DATABASE_URI"); envDB != "" {
		cfg.DatabaseURI = envDB
	}

	if envAccrual := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrual != "" {
		cfg.AccrualSystemAddress = envAccrual
	}

	if envSecret := os.Getenv("JWT_SECRET"); envSecret != "" {
		cfg.JWTSecret = envSecret
	}

	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "gophermart-default-secret-key"
	}

	cfg.AccrualWorkers = 5
	if envWorkers := os.Getenv("ACCRUAL_WORKERS"); envWorkers != "" {
		if val, err := strconv.Atoi(envWorkers); err == nil {
			cfg.AccrualWorkers = val
		}
	}

	cfg.AccrualMaxConcurrency = 5
	if envMaxConcurrency := os.Getenv("ACCRUAL_MAX_CONCURRENCY"); envMaxConcurrency != "" {
		if val, err := strconv.Atoi(envMaxConcurrency); err == nil {
			cfg.AccrualMaxConcurrency = val
		}
	}

	cfg.AccrualCheckInterval = 1 * time.Second
	if envInterval := os.Getenv("ACCRUAL_CHECK_INTERVAL"); envInterval != "" {
		if val, err := time.ParseDuration(envInterval); err == nil {
			cfg.AccrualCheckInterval = val
		}
	}

	return cfg, nil
}
