package accrual

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"go-musthave-diploma/internal/logger"
	"go-musthave-diploma/internal/models"
	"go-musthave-diploma/internal/storage"

	"golang.org/x/sync/semaphore"
)

type Worker struct {
	id            int
	storage       storage.OrderRepository
	accrualClient *AccrualClient
	semaphore     *semaphore.Weighted
}

func NewWorker(id int, storage storage.OrderRepository, client *AccrualClient, sem *semaphore.Weighted) *Worker {
	return &Worker{
		id:            id,
		storage:       storage,
		accrualClient: client,
		semaphore:     sem,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	logger.Log.Infow("Worker started", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			logger.Log.Infow("Worker stopped", "worker_id", w.id)
			return
		case <-ticker.C:
			w.processNextOrder(ctx)
		}
	}
}

func (w *Worker) processNextOrder(ctx context.Context) {
	order, err := w.storage.GetNextOrderForProcessing(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrOrderNotFound) {
			return
		}
		logger.Log.Errorw("Failed to get next order", "worker_id", w.id, "error", err)
		return
	}

	logger.Log.Infow("Processing order", "worker_id", w.id, "order_number", order.Number)

	if err := w.semaphore.Acquire(ctx, 1); err != nil {
		logger.Log.Warnw("Failed to acquire semaphore", "worker_id", w.id, "error", err)
		return
	}
	defer w.semaphore.Release(1)

	if err := w.checkAndUpdateOrder(ctx, order); err != nil {
		logger.Log.Errorw("Failed to process order", "worker_id", w.id, "order_number", order.Number, "error", err)
	}
}

func (w *Worker) checkAndUpdateOrder(ctx context.Context, order *models.Order) error {
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			logger.Log.Infow("Retrying after backoff", "worker_id", w.id, "order_number", order.Number, "attempt", attempt, "backoff", backoff)
			time.Sleep(backoff)
		}

		resp, statusCode, err := w.accrualClient.GetOrder(ctx, order.Number)

		if err != nil && statusCode == http.StatusTooManyRequests {
			logger.Log.Warnw("Rate limited, waiting", "worker_id", w.id, "order_number", order.Number)
			time.Sleep(60 * time.Second)
			continue
		}

		if err != nil && statusCode == http.StatusInternalServerError {
			logger.Log.Warnw("Accrual server error", "worker_id", w.id, "order_number", order.Number, "attempt", attempt+1)
			continue
		}

		if err != nil {
			logger.Log.Errorw("Failed to get order from accrual", "worker_id", w.id, "order_number", order.Number, "error", err)
			return err
		}

		if statusCode == http.StatusNoContent {
			logger.Log.Infow("Order not registered in accrual", "worker_id", w.id, "order_number", order.Number)
			return nil
		}

		if resp != nil {
			return w.updateOrderFromResponse(ctx, order.Number, resp)
		}

		return nil
	}

	logger.Log.Warnw("Max retries exceeded", "worker_id", w.id, "order_number", order.Number)
	return errors.New("max retries exceeded")
}

func (w *Worker) updateOrderFromResponse(ctx context.Context, number string, resp *AccrualResponse) error {
	var newStatus string

	switch resp.Status {
	case "REGISTERED":
		newStatus = models.OrderStatusProcessing
	case "PROCESSING":
		newStatus = models.OrderStatusProcessing
	case "INVALID":
		newStatus = models.OrderStatusInvalid
	case "PROCESSED":
		newStatus = models.OrderStatusProcessed
	default:
		logger.Log.Warnw("Unknown status from accrual", "status", resp.Status, "order_number", number)
		return nil
	}

	err := w.storage.UpdateOrderStatus(ctx, number, newStatus, resp.Accrual)
	if err != nil {
		return err
	}

	logger.Log.Infow("Order updated", "worker_id", w.id, "order_number", number, "status", newStatus, "accrual", resp.Accrual)
	return nil
}
