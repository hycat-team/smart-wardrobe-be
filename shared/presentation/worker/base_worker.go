package worker

import (
	"context"
	"sync/atomic"
)

type BaseWorker struct {
	name         string
	workerCtx    context.Context
	cancel       context.CancelFunc
	running      atomic.Bool
	restartCount atomic.Int32
	healthFn     func(context.Context) error
}

func NewBaseWorker(name string) *BaseWorker {
	return &BaseWorker{name: name}
}

func (w *BaseWorker) Start(ctx context.Context) {
	if w.running.Load() {
		return
	}
	w.workerCtx, w.cancel = context.WithCancel(ctx)
	w.running.Store(true)
}

// Context trả về ctx derive từ Start — bị cancel khi Stop().
// Worker PHẢI dùng ctx này cho goroutine consume/loop để graceful
// shutdown hoạt động đúng (trước đây ctx derive bị vứt nên consumer
// chạy dưới context.Background() vĩnh viễn, Stop() không có tác dụng).
func (w *BaseWorker) Context() context.Context {
	if w.workerCtx == nil {
		return context.Background()
	}
	return w.workerCtx
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

func (w *BaseWorker) Name() string {
	return w.name
}

func (w *BaseWorker) SetHealthCheck(fn func(context.Context) error) {
	w.healthFn = fn
}
