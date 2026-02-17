package supervisor

import "context"

type Worker struct {
	ID       string
	restarts int
	ctx      context.Context
	cancel   context.CancelFunc
	handler  func(context.Context) error
}
