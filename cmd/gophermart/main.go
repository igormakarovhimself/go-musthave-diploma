package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	defer logger.Log.Sync()

	db, err := storage.InitDB(cfg.DatabaseURI)
	if err != nil {
		return err
	}
	defer db.Close()

	userRepo := storage.NewPostgresStorage(db)

	logger.Log.Infow("Starting Gophermart",
		"address", cfg.RunAddress,
	)

	router := app.NewRouter(userRepo, cfg.JWTSecret)

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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Log.Errorw("Server shutdown error", "error", err)
		}

		close(idleConnsClosed)
	}()

	logger.Log.Infow("Server started", "address", cfg.RunAddress)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	logger.Log.Info("Server stopped")

	return nil
}
