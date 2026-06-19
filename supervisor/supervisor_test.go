package supervisor_test

import (
	"testing"
	"time"

	"my-app/actor"
	"my-app/supervisor"
)

type MockCrashActor struct{}

func (a *MockCrashActor) Run(mailbox chan actor.Message) error {
	for msg := range mailbox {
		if msg.Type == "panic" {
			panic("simulated mock panic")
		}
		if msg.ResponseChan != nil {
			msg.ResponseChan <- actor.Response{Data: "ok", Err: nil}
		}
	}
	return nil
}

func TestSupervisorRestartOnPanic(t *testing.T) {
	registry := actor.NewActorRegistry()
	gateway := actor.NewActorGateway(registry)

	sup := supervisor.NewSupervisor(registry, func() actor.Actor {
		return &MockCrashActor{}
	})
	
	// Fast retries for testing
	sup.WithPolicy(supervisor.RestartMaxRetries, 3, 10*time.Millisecond, 50*time.Millisecond)
	sup.Start()
	defer sup.Stop()

	// Wait for actor to register
	time.Sleep(50 * time.Millisecond)

	// Send normal message
	res, err := gateway.Send("hello", nil, 100*time.Millisecond)
	if err != nil || res != "ok" {
		t.Fatalf("Expected 'ok' response, got %v, err: %v", res, err)
	}

	// Verify restart count is 0
	restarts, status := sup.GetStats()
	if restarts != 0 || status != "running" {
		t.Fatalf("Expected 0 restarts, got %d, status: %s", restarts, status)
	}

	// Trigger panic
	_, _ = gateway.Send("panic", nil, 50*time.Millisecond)

	// Wait for supervisor to catch and restart
	time.Sleep(100 * time.Millisecond)

	// Verify restart count is 1
	restarts, status = sup.GetStats()
	if restarts != 1 || status != "running" {
		t.Fatalf("Expected 1 restart, got %d, status: %s", restarts, status)
	}

	// Send normal message again to verify actor is recovered
	res, err = gateway.Send("hello", nil, 100*time.Millisecond)
	if err != nil || res != "ok" {
		t.Fatalf("Expected 'ok' response after recovery, got %v, err: %v", res, err)
	}
}
