package supervisor

import (
	"context"
	"log"
	"sync"
	"time"
)

type Supervisor struct {
	workers []*Worker
	mutex   sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewSupervisor() *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Supervisor{
		workers: make([]*Worker, 0),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (s *Supervisor) AddWorker(id string, handler func(context.Context) error) {
	ctx, cancel := context.WithCancel(s.ctx)
	worker := &Worker{
		ID:      id,
		ctx:     ctx,
		cancel:  cancel,
		handler: handler,
	}
	s.mutex.Lock()
	s.workers = append(s.workers, worker)
	s.mutex.Unlock()

	go s.monitorWorker(worker)
}

func (s *Supervisor) Stop() {
	s.cancel()
}

func (s *Supervisor) monitorWorker(worker *Worker) {
	for {
		select {
		case <-worker.ctx.Done():
			return
		default:
			err := worker.handler(worker.ctx)
			if err != nil {
				log.Printf("Worker %s failed: %v", worker.ID, err)
				worker.restarts++

				if worker.restarts > 10 {
					log.Printf("Worker %s exceeded restart limit", worker.ID)
					return
				}

				backoff := time.Duration(worker.restarts) * time.Second
				log.Printf("Restarting worker %s in %v", worker.ID, backoff)
				time.Sleep(backoff)
			}
		}
	}
}
