package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-musthave-diploma/internal/accrual"
	"go-musthave-diploma/internal/app"
	"go-musthave-diploma/internal/config"
	"go-musthave-diploma/internal/logger"
	"go-musthave-diploma/internal/storage"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := logger.Initialize("info"); err != nil {
		return err
	}

	db, err := storage.InitDB(cfg.DatabaseURI)
	if err != nil {
		return err
	}
	defer db.Close()

	store := storage.NewPostgresStorage(db)

	var processor *accrual.OrderProcessor
	if cfg.AccrualSystemAddress != "" {
		processor = accrual.NewOrderProcessor(store, cfg.AccrualSystemAddress, cfg.AccrualWorkers, cfg.AccrualMaxConcurrency)
		ctx := context.Background()
		processor.Start(ctx)
		logger.Log.Info("Order processor started", "workers", cfg.AccrualWorkers, "max_concurrency", cfg.AccrualMaxConcurrency, "accrual_address", cfg.AccrualSystemAddress)
	} else {
		logger.Log.Warn("Accrual system address not configured, order processing disabled")
	}

	logger.Log.Info("Starting Gophermart",
		"address", cfg.RunAddress,
	)

	router := app.NewRouter(store, store, store, store, cfg.JWTSecret)

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	idleConnsClosed := make(chan struct{})
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigint
		logger.Log.Info("Shutting down server...")

		if processor != nil {
			processor.Shutdown()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Log.Error("Server shutdown error", "error", err)
		}

		close(idleConnsClosed)
	}()

	logger.Log.Info("Server started", "address", cfg.RunAddress)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	logger.Log.Info("Server stopped")

	return nil
}
