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

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		RunAddress:            "localhost:8080",
		JWTSecret:             "gophermart-default-secret-key",
		AccrualWorkers:        5,
		AccrualMaxConcurrency: 5,
		AccrualCheckInterval:  1 * time.Second,
	}
}

func New(opts ...Option) *Config {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func WithRunAddress(addr string) Option {
	return func(c *Config) {
		if addr != "" {
			c.RunAddress = addr
		}
	}
}

func WithDatabaseURI(uri string) Option {
	return func(c *Config) {
		c.DatabaseURI = uri
	}
}

func WithAccrualSystemAddress(addr string) Option {
	return func(c *Config) {
		c.AccrualSystemAddress = addr
	}
}

func WithJWTSecret(secret string) Option {
	return func(c *Config) {
		if secret != "" {
			c.JWTSecret = secret
		}
	}
}

func WithAccrualWorkers(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.AccrualWorkers = n
		}
	}
}

func WithAccrualMaxConcurrency(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.AccrualMaxConcurrency = n
		}
	}
}

func WithAccrualCheckInterval(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.AccrualCheckInterval = d
		}
	}
}

func Load() (*Config, error) {
	var runAddress, databaseURI, accrualAddr, jwtSecret string

	flag.StringVar(&runAddress, "a", "", "HTTP server address")
	flag.StringVar(&databaseURI, "d", "", "PostgreSQL connection URI")
	flag.StringVar(&accrualAddr, "r", "", "Accrual system address")
	flag.StringVar(&jwtSecret, "j", "", "JWT secret key")

	flag.Parse()

	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		runAddress = envAddr
	}
	if envDB := os.Getenv("DATABASE_URI"); envDB != "" {
		databaseURI = envDB
	}
	if envAccrual := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrual != "" {
		accrualAddr = envAccrual
	}
	if envSecret := os.Getenv("JWT_SECRET"); envSecret != "" {
		jwtSecret = envSecret
	}

	opts := []Option{
		WithRunAddress(runAddress),
		WithDatabaseURI(databaseURI),
		WithAccrualSystemAddress(accrualAddr),
		WithJWTSecret(jwtSecret),
	}

	if envWorkers := os.Getenv("ACCRUAL_WORKERS"); envWorkers != "" {
		if val, err := strconv.Atoi(envWorkers); err == nil {
			opts = append(opts, WithAccrualWorkers(val))
		}
	}
	if envMaxConcurrency := os.Getenv("ACCRUAL_MAX_CONCURRENCY"); envMaxConcurrency != "" {
		if val, err := strconv.Atoi(envMaxConcurrency); err == nil {
			opts = append(opts, WithAccrualMaxConcurrency(val))
		}
	}
	if envInterval := os.Getenv("ACCRUAL_CHECK_INTERVAL"); envInterval != "" {
		if val, err := time.ParseDuration(envInterval); err == nil {
			opts = append(opts, WithAccrualCheckInterval(val))
		}
	}

	return New(opts...), nil
}
