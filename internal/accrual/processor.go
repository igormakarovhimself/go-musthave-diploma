package accrual

import (
	"context"
	"sync"

	"go-musthave-diploma/internal/logger"
	"go-musthave-diploma/internal/storage"
)

type OrderProcessor struct {
	workers       []*Worker
	wg            *sync.WaitGroup
	accrualClient *AccrualClient
	storage       storage.OrderRepository
	numWorkers    int
	cancel        context.CancelFunc
}

func NewOrderProcessor(storage storage.OrderRepository, accrualURL string, numWorkers int) *OrderProcessor {
	client := NewAccrualClient(accrualURL)
	semaphore := NewSemaphore(3)

	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(i+1, storage, client, semaphore)
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
