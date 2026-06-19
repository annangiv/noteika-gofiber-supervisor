package actor

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrActorUnavailable = errors.New("supervised actor is currently unavailable")
	ErrRequestTimeout   = errors.New("request to actor timed out")
)

// ActorRegistry holds the mailbox channel for the currently active instance of an actor.
// It is updated by the supervisor when an actor restarts.
type ActorRegistry struct {
	mu      sync.RWMutex
	mailbox chan Message
}

func NewActorRegistry() *ActorRegistry {
	return &ActorRegistry{}
}

// Register stores the active actor's channel.
func (r *ActorRegistry) Register(mailbox chan Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mailbox = mailbox
}

// Unregister clears the actor's channel.
func (r *ActorRegistry) Unregister() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mailbox = nil
}

// GetMailbox retrieves the active channel.
func (r *ActorRegistry) GetMailbox() chan Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mailbox
}

// ActorGateway is the entry point for other packages to communicate with supervised actors.
// It resolves the actor mailbox dynamically, so callers do not fail when the actor is restarting.
type ActorGateway struct {
	registry *ActorRegistry
}

func NewActorGateway(registry *ActorRegistry) *ActorGateway {
	return &ActorGateway{registry: registry}
}

// Send sends a message to the actor and waits for a response or timeout.
func (g *ActorGateway) Send(msgType string, payload interface{}, timeout time.Duration) (interface{}, error) {
	mailbox := g.registry.GetMailbox()
	if mailbox == nil {
		return nil, ErrActorUnavailable
	}

	responseChan := make(chan Response, 1)
	msg := Message{
		Type:         msgType,
		Payload:      payload,
		ResponseChan: responseChan,
	}

	// Send message to mailbox channel (buffered)
	select {
	case mailbox <- msg:
	case <-time.After(timeout):
		return nil, ErrRequestTimeout
	}

	// Wait for response
	select {
	case res := <-responseChan:
		return res.Data, res.Err
	case <-time.After(timeout):
		return nil, ErrRequestTimeout
	}
}
