package worker

import (
	"context"
	"sync/atomic"
)

type BaseWorker struct {
	name         string
	cancel       context.CancelFunc
	running      atomic.Bool
	restartCount int32
	healthFn     func(context.Context) error
}

func NewBaseWorker(name string) *BaseWorker {
	return &BaseWorker{name: name}
}

func (w *BaseWorker) Start(ctx context.Context) {
	if w.running.Load() {
		return
	}
	ctx, w.cancel = context.WithCancel(ctx)
	w.running.Store(true)
}

func (w *BaseWorker) Stop() {
	if !w.running.Load() {
		return
	}
	if w.cancel != nil {
		w.cancel()
	}
	w.running.Store(false)
}

func (w *BaseWorker) IsRunning() bool {
	return w.running.Load()
}

func (w *BaseWorker) Name() string {
	return w.name
}

func (w *BaseWorker) RestartCount() int32 {
	return atomic.LoadInt32(&w.restartCount)
}

func (w *BaseWorker) SetHealthCheck(fn func(context.Context) error) {
	w.healthFn = fn
}
