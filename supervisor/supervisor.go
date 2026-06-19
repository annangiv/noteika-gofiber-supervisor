package supervisor

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"my-app/actor"
)

type RestartPolicy string

const (
	RestartAlways     RestartPolicy = "always"
	RestartNever      RestartPolicy = "never"
	RestartMaxRetries RestartPolicy = "max_retries"
)

type Supervisor struct {
	registry *actor.ActorRegistry
	factory  func() actor.Actor

	policy      RestartPolicy
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration

	attempts int
	restarts int
	status   string

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

func NewSupervisor(registry *actor.ActorRegistry, factory func() actor.Actor) *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Supervisor{
		registry:    registry,
		factory:     factory,
		policy:      RestartMaxRetries,
		maxAttempts: 5,
		baseDelay:   100 * time.Millisecond,
		maxDelay:    5 * time.Second,
		status:      "stopped",
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (s *Supervisor) WithPolicy(policy RestartPolicy, maxAttempts int, baseDelay, maxDelay time.Duration) *Supervisor {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = policy
	s.maxAttempts = maxAttempts
	s.baseDelay = baseDelay
	s.maxDelay = maxDelay
	return s
}

func (s *Supervisor) Start() {
	s.mu.Lock()
	s.status = "running"
	s.mu.Unlock()

	go s.monitor()
}

func (s *Supervisor) Stop() {
	s.cancel()
	s.mu.Lock()
	s.status = "stopped"
	s.registry.Unregister()
	s.mu.Unlock()
	log.Println("[Supervisor] Stopped monitoring")
}

func (s *Supervisor) GetStats() (int, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.restarts, s.status
}

func (s *Supervisor) monitor() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			mailbox := make(chan actor.Message, 100)
			s.registry.Register(mailbox)

			// Update status to running
			s.mu.Lock()
			s.status = "running"
			s.mu.Unlock()

			// Run child actor in a supervised wrapper
			err := s.runActor(mailbox)

			// Clean up registry since actor stopped/crashed
			s.registry.Unregister()
			close(mailbox)

			select {
			case <-s.ctx.Done():
				return
			default:
			}

			if err != nil {
				log.Printf("[Supervisor] Child actor exited with error: %v", err)
			} else {
				log.Println("[Supervisor] Child actor exited cleanly")
			}

			if !s.shouldRetry() {
				s.mu.Lock()
				s.status = "failed"
				s.mu.Unlock()
				log.Println("[Supervisor] Max restart attempts reached or policy is Never. Giving up.")
				return
			}

			s.mu.Lock()
			s.attempts++
			s.restarts++
			s.status = "restarting"
			s.mu.Unlock()

			delay := s.calculateDelay()
			log.Printf("[Supervisor] Restarting child actor (attempt %d) in %v...", s.attempts, delay)

			select {
			case <-s.ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}
}

func (s *Supervisor) runActor(mailbox chan actor.Message) (err error) {
	// Intercept panics and return them as standard errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("actor panic: %v", r)
			log.Printf("[Supervisor] Caught panic from child actor: %v", r)
		}
	}()

	// Instantiates the actor and runs it
	child := s.factory()
	return child.Run(mailbox)
}

func (s *Supervisor) shouldRetry() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.policy {
	case RestartAlways:
		return true
	case RestartNever:
		return false
	case RestartMaxRetries:
		return s.attempts < s.maxAttempts
	default:
		return false
	}
}

func (s *Supervisor) calculateDelay() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.policy == RestartAlways {
		return s.baseDelay
	}

	// Exponential backoff: baseDelay * 2^(attempts-1)
	exponent := float64(s.attempts - 1)
	factor := math.Pow(2, exponent)
	delay := time.Duration(float64(s.baseDelay) * factor)

	if delay > s.maxDelay {
		return s.maxDelay
	}
	return delay
}
