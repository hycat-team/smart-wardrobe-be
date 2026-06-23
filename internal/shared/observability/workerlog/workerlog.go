package workerlog

import (
	"time"

	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	MessageReceived     = "Worker message received"
	MessageRunSucceeded = "Worker run succeeded"
	MessageRunFailed    = "Worker run failed"
	TriggerStartup      = "startup"
	TriggerCron         = "cron"
	TriggerQueue        = "queue"
	TriggerRequeue      = "requeue"
	TriggerInitialSync  = "initial_sync"
	ResultSuccess       = "success"
	ResultFailed        = "failed"
)

type Counters struct {
	TotalCount   int
	SuccessCount int
	WarningCount int
	ErrorCount   int
	SkippedCount int
	RetryCount   int
}

type Run struct {
	worker      string
	triggerType string
	runID       string
	startedAt   time.Time
	counters    Counters
	summary     []zap.Field
}

func New(worker string, triggerType string) *Run {
	return &Run{
		worker:      worker,
		triggerType: triggerType,
		runID:       uuid.NewString(),
		startedAt:   time.Now().UTC(),
	}
}

func (r *Run) Worker() string {
	return r.worker
}

func (r *Run) TriggerType() string {
	return r.triggerType
}

func (r *Run) RunID() string {
	return r.runID
}

func (r *Run) BaseFields(extra ...zap.Field) []zap.Field {
	fields := []zap.Field{
		zap.String("worker", r.worker),
		zap.String("triggerType", r.triggerType),
		zap.String("runId", r.runID),
	}
	return append(fields, extra...)
}

func (r *Run) ChildWarn(log logger.Interface, message string, extra ...zap.Field) {
	r.counters.WarningCount++
	log.Warn(message, r.BaseFields(extra...)...)
}

func (r *Run) ChildError(log logger.Interface, message string, extra ...zap.Field) {
	r.counters.ErrorCount++
	log.Error(message, r.BaseFields(extra...)...)
}

func (r *Run) ChildInfo(log logger.Interface, message string, extra ...zap.Field) {
	log.Info(message, r.BaseFields(extra...)...)
}

func (r *Run) LogReceived(log logger.Interface, extra ...zap.Field) {
	log.Info(MessageReceived, r.BaseFields(extra...)...)
}

func (r *Run) AddTotal(delta int) {
	r.counters.TotalCount += delta
}

func (r *Run) AddSuccess(delta int) {
	r.counters.SuccessCount += delta
}

func (r *Run) AddWarning(delta int) {
	r.counters.WarningCount += delta
}

func (r *Run) AddError(delta int) {
	r.counters.ErrorCount += delta
}

func (r *Run) AddSkipped(delta int) {
	r.counters.SkippedCount += delta
}

func (r *Run) AddRetry(delta int) {
	r.counters.RetryCount += delta
}

func (r *Run) Counters() Counters {
	return r.counters
}

func (r *Run) AddSummaryFields(fields ...zap.Field) {
	r.summary = append(r.summary, fields...)
}

func (r *Run) SummaryFields(result string, extra ...zap.Field) []zap.Field {
	fields := r.BaseFields(
		zap.Int64("durationMs", time.Since(r.startedAt).Milliseconds()),
		zap.String("result", result),
		zap.Int("totalCount", r.counters.TotalCount),
		zap.Int("successCount", r.counters.SuccessCount),
		zap.Int("warningCount", r.counters.WarningCount),
		zap.Int("errorCount", r.counters.ErrorCount),
		zap.Int("skippedCount", r.counters.SkippedCount),
		zap.Int("retryCount", r.counters.RetryCount),
	)
	fields = append(fields, r.summary...)
	return append(fields, extra...)
}

func (r *Run) LogSuccess(log logger.Interface, extra ...zap.Field) {
	log.Info(MessageRunSucceeded, r.SummaryFields(ResultSuccess, extra...)...)
}

func (r *Run) LogFailure(log logger.Interface, err error, extra ...zap.Field) {
	fields := extra
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	log.Error(MessageRunFailed, r.SummaryFields(ResultFailed, fields...)...)
}
