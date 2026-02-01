package accrual

import (
	"context"
	"sync"

	"go-musthave-diploma/internal/logger"
	"go-musthave-diploma/internal/storage"

	"golang.org/x/sync/semaphore"
)

type OrderProcessor struct {
	workers       []*Worker
	wg            *sync.WaitGroup
	accrualClient *AccrualClient
	storage       storage.OrderRepository
	numWorkers    int
	cancel        context.CancelFunc
}

func NewOrderProcessor(storage storage.OrderRepository, accrualURL string, numWorkers int, maxConcurrency int) *OrderProcessor {
	client := NewAccrualClient(accrualURL)
	sem := semaphore.NewWeighted(int64(maxConcurrency))

	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(i+1, storage, client, sem)
	}

	return &OrderProcessor{
		workers:       workers,
		wg:            &sync.WaitGroup{},
		accrualClient: client,
		storage:       storage,
		numWorkers:    numWorkers,
	}
}

func (p *OrderProcessor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	for _, worker := range p.workers {
		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			w.Run(ctx)
		}(worker)
	}

	logger.Log.Infow("Order processor started", "num_workers", p.numWorkers)
}

func (p *OrderProcessor) Shutdown() {
	logger.Log.Info("Shutting down order processor...")

	if p.cancel != nil {
		p.cancel()
	}

	p.wg.Wait()

	logger.Log.Info("Order processor stopped")
}
