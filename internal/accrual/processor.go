package accrual

import (
	"context"

	"go-musthave-diploma/internal/logger"
	"go-musthave-diploma/internal/storage"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type OrderProcessor struct {
	workers       []*Worker
	accrualClient *AccrualClient
	storage       storage.OrderRepository
	numWorkers    int
	group         *errgroup.Group
	groupCtx      context.Context
	cancelFunc    context.CancelFunc
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
		accrualClient: client,
		storage:       storage,
		numWorkers:    numWorkers,
	}
}

func (p *OrderProcessor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel

	g, gCtx := errgroup.WithContext(ctx)
	p.group = g
	p.groupCtx = gCtx

	for _, worker := range p.workers {
		w := worker
		g.Go(func() error {
			w.Run(gCtx)
			return nil
		})
	}

	logger.Log.Infow("Order processor started", "num_workers", p.numWorkers)
}

func (p *OrderProcessor) Shutdown() {
	logger.Log.Info("Shutting down order processor...")

	if p.cancelFunc != nil {
		p.cancelFunc()
	}

	if p.group != nil {
		if err := p.group.Wait(); err != nil {
			logger.Log.Errorw("Error during processor shutdown", "error", err)
		}
	}

	logger.Log.Info("Order processor stopped")
}
