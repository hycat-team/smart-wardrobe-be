package grant

import (
	"os"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// capturedLogger keeps the emitted log entries so a test can assert on the
// WARNING level, not merely on the counter.
//
// The level matters: SC-023 requires a rejected event to raise an alert, and a
// counter nobody reads satisfies the letter of the requirement while leaving the
// customer without an entitlement. The assertion is therefore on what an
// operator would actually see.
type capturedLogger struct {
	mu       sync.Mutex
	core     zapcore.Core
	observed *observer.ObservedLogs
}

func newCapturedLogger() *capturedLogger {
	core, logs := observer.New(zapcore.DebugLevel)
	return &capturedLogger{core: core, observed: logs}
}

func (l *capturedLogger) Debug(msg string, fields ...zap.Field) {
	l.write(zapcore.DebugLevel, msg, fields)
}
func (l *capturedLogger) Info(msg string, fields ...zap.Field) {
	l.write(zapcore.InfoLevel, msg, fields)
}
func (l *capturedLogger) Warn(msg string, fields ...zap.Field) {
	l.write(zapcore.WarnLevel, msg, fields)
}
func (l *capturedLogger) Error(msg string, fields ...zap.Field) {
	l.write(zapcore.ErrorLevel, msg, fields)
}
func (l *capturedLogger) Fatal(msg string, fields ...zap.Field) {
	l.write(zapcore.FatalLevel, msg, fields)
}

// count reports how many entries were logged at a level.
func (l *capturedLogger) count(level zapcore.Level) int {
	return l.observed.FilterLevelExact(level).Len()
}

func (l *capturedLogger) write(level zapcore.Level, msg string, fields []zap.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := zapcore.Entry{Level: level, Message: msg}
	l.core.Write(entry, fields)
}

// SC-023: a rejected event raises a WARNING carrying the reason code and a
// running total, so an operator can see both what happened and how often.
func TestGrantMetricsRejectedEventWarnsWithReasonAndTotal(t *testing.T) {
	log := newCapturedLogger()
	metrics := NewGrantMetrics(log)

	metrics.RecordRejection(ReasonUnknownEventType, "11111111-1111-1111-1111-111111111111")
	metrics.RecordRejection(ReasonUnknownEventType, "11111111-1111-1111-1111-111111111111")

	entries := log.observed.All()
	if len(entries) != 2 {
		t.Fatalf("phải ghi một cảnh báo cho mỗi lần từ chối, có = %d", len(entries))
	}
	for _, entry := range entries {
		if entry.Level != zapcore.WarnLevel {
			t.Errorf("mức cảnh báo phải là warn, có = %s", entry.Level)
		}
		fields := entry.ContextMap()
		if fields["reason"] != ReasonUnknownEventType {
			t.Errorf("cảnh báo phải nêu mã lý do, có = %v", fields["reason"])
		}
		if _, ok := fields["rejected_total_for_reason"]; !ok {
			t.Error("cảnh báo phải kèm tổng số lần bị từ chối theo mã lý do")
		}
	}

	if got := metrics.Snapshot()[ReasonUnknownEventType]; got != 2 {
		t.Errorf("bộ đếm phải bằng 2, có = %d", got)
	}
}

// FR-087: counters are kept PER REASON. A single total would answer "was anything
// rejected?" but not "which cause?", and the cause is what determines the fix.
func TestGrantMetricsCountsAreKeyedByReason(t *testing.T) {
	metrics := NewGrantMetrics(newCapturedLogger())

	metrics.RecordRejection(ReasonUnknownEventType, "11111111-1111-1111-1111-111111111111")
	metrics.RecordRejection(ReasonMalformedTimestamp, "11111111-1111-1111-1111-111111111111")
	metrics.RecordRejection(ReasonMalformedTimestamp, "11111111-1111-1111-1111-111111111111")
	metrics.RecordRejection(ReasonMissingSourceEventID, "11111111-1111-1111-1111-111111111111")

	snapshot := metrics.Snapshot()
	want := map[string]int64{
		ReasonUnknownEventType:     1,
		ReasonMalformedTimestamp:   2,
		ReasonMissingSourceEventID: 1,
	}
	if len(snapshot) != len(want) {
		t.Fatalf("phải có đúng %d mã lý do, có = %d (%v)", len(want), len(snapshot), snapshot)
	}
	for reason, expected := range want {
		if snapshot[reason] != expected {
			t.Errorf("mã %s: mong đợi %d, có = %d", reason, expected, snapshot[reason])
		}
	}
}

// A duplicate is the normal consequence of at-least-once delivery, so it is
// logged one level below a genuine refusal. Both are counted; only the level
// differs.
func TestGrantMetricsDuplicateIsCountedButLoggedLower(t *testing.T) {
	log := newCapturedLogger()
	metrics := NewGrantMetrics(log)

	metrics.RecordRejection(ReasonDuplicateSourceEvent, "11111111-1111-1111-1111-111111111111")

	entries := log.observed.All()
	if len(entries) != 1 {
		t.Fatalf("phải ghi một dòng cho sự kiện trùng, có = %d", len(entries))
	}
	if entries[0].Level != zapcore.InfoLevel {
		t.Errorf("sự kiện trùng là hệ quả bình thường của giao hóa at-least-once, phải ở mức info, có = %s", entries[0].Level)
	}
	if got := metrics.Snapshot()[ReasonDuplicateSourceEvent]; got != 1 {
		t.Errorf("sự kiện trùng vẫn phải được đếm, có = %d", got)
	}
}

// The counters are read by monitoring while the consumer writes them.
func TestGrantMetricsIsSafeForConcurrentRecordAndSnapshot(t *testing.T) {
	metrics := NewGrantMetrics(newCapturedLogger())

	var wg sync.WaitGroup
	for writer := 0; writer < 8; writer++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				metrics.RecordRejection(ReasonUnknownEventType, "11111111-1111-1111-1111-111111111111")
				metrics.Snapshot()
			}
		}()
	}
	wg.Wait()

	if got := metrics.Snapshot()[ReasonUnknownEventType]; got != 800 {
		t.Errorf("mọi lần ghi phải được đếm, mong đợi 800, có = %d", got)
	}
}

// T114/SC-025: the rejection line must answer "which account, which reason, at
// what moment" on its own.
//
// A line carrying only the reason forces an operator to correlate against some
// other log stream at exactly the moment nobody is watching, which is the whole
// moment observability exists for. The field is named user_id so it joins with
// every other line in this service.
func TestGrantMetricsRejectionLineCarriesTheAccount(t *testing.T) {
	log := newCapturedLogger()
	metrics := NewGrantMetrics(log)
	const userID = "ab11cd11-ab11-ab11-ab11-ab11ab11ab11"

	metrics.RecordRejection(ReasonUnknownEventType, userID)

	entries := log.observed.All()
	if len(entries) != 1 {
		t.Fatalf("phải ghi đúng một dòng, có = %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["user_id"] != userID {
		t.Errorf("dòng cảnh báo phải nêu tài khoản, có = %v", fields["user_id"])
	}
	if fields["reason"] != ReasonUnknownEventType {
		t.Errorf("dòng cảnh báo phải nêu mã lý do, có = %v", fields["reason"])
	}
	if _, ok := fields["at"]; !ok {
		t.Error("dòng cảnh báo phải nêu thời điểm")
	}
}

// A payload too malformed to parse may carry no usable id. The field must still
// be PRESENT — an absent field breaks the log query, an empty one does not.
func TestGrantMetricsRejectionLineKeepsTheFieldWhenUserIsUnusable(t *testing.T) {
	log := newCapturedLogger()
	metrics := NewGrantMetrics(log)

	metrics.RecordRejection(ReasonMalformedTimestamp, "")

	entries := log.observed.All()
	if len(entries) != 1 {
		t.Fatalf("phải ghi đúng một dòng, có = %d", len(entries))
	}
	fields := entries[0].ContextMap()
	value, present := fields["user_id"]
	if !present {
		t.Fatal("trường user_id phải luôn hiện diện, kể cả khi rỗng")
	}
	if value != "" {
		t.Errorf("tài khoản không đọc được phải để trống, có = %v", value)
	}
}

// Snapshot hands back a COPY. A caller that mutated the returned map would
// corrupt the counters of a running service, and the compiler cannot stop it.
func TestGrantMetricsSnapshotReturnsACopy(t *testing.T) {
	metrics := NewGrantMetrics(newCapturedLogger())
	metrics.RecordRejection(ReasonUnknownEventType, "11111111-1111-1111-1111-111111111111")

	snapshot := metrics.Snapshot()
	snapshot[ReasonUnknownEventType] = 9999
	snapshot["injected"] = 1

	if got := metrics.Snapshot()[ReasonUnknownEventType]; got != 1 {
		t.Errorf("thay đổi bản sao không được ảnh hưởng bộ đếm, có = %d", got)
	}
	if _, injected := metrics.Snapshot()["injected"]; injected {
		t.Error("mã lý do bị chèn từ bản sao không được xuất hiện trong bộ đếm gốc")
	}
}

// ---------------------------------------------------------------------------
// 024 T175 — the comments must describe the mechanism that actually runs.
// ---------------------------------------------------------------------------

// A comment claiming a mechanism that does not exist is not cosmetic. This one said
// the observer is "backed by a Prometheus gauge", and there is no Prometheus anywhere
// in the repository — not in this package, not in any go.mod. A reader who believed it
// would assume the counters are scrapeable and durable, and would not think to wire
// log alerting, which is the alerting that actually exists.
func TestMetricsCommentsDoNotClaimAMetricsBackendThatDoesNotExist(t *testing.T) {
	for _, file := range []string{"grant_reject_metrics.go", "grant_fallback_resolver.go"} {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("đọc %s: %v", file, err)
		}
		text := strings.ToLower(string(source))

		if strings.Contains(text, "prometheus") {
			// The corrected comments mention the word only to say it is absent. Any
			// claim that something IS backed by it is the regression.
			for _, claim := range []string{
				"backed by a prometheus",
				"prometheus gauge whose",
			} {
				if strings.Contains(text, claim) {
					t.Errorf("%s khẳng định cơ chế Prometheus không tồn tại: %q", file, claim)
				}
			}
		}
	}
}

// FR-087 asks for a counter per reason and for an alert when it rises. The honest
// account of what runs is: process-local counters, exposed on the health response,
// with one log line per rejection and NO numeric threshold. This pins the parts that
// are real, so the documentation cannot quietly claim a threshold either.
func TestRejectionAlertingIsLogBasedWithNoInventedThreshold(t *testing.T) {
	log := newCapturedLogger()
	metrics := NewGrantMetrics(log)

	metrics.RecordRejection(ReasonUnknownEventType, "user-1")

	if log.count(zapcore.WarnLevel) != 1 {
		t.Errorf("mỗi lần từ chối phải phát đúng một cảnh báo, có = %d", log.count(zapcore.WarnLevel))
	}

	// The counter is real and readable, which is what makes the warning more than a
	// log line: bootstrap/router.go serves this snapshot on the health endpoint.
	snapshot := metrics.Snapshot()
	if snapshot[ReasonUnknownEventType] != 1 {
		t.Errorf("bộ đếm phải đếm được, có = %d", snapshot[ReasonUnknownEventType])
	}
}

// The counter is process-local. Asserting that explicitly is what stops someone later
// reading "counter" and assuming durability across a restart, which is the gap against
// FR-087 that the comment now names.
func TestRejectionCounterIsProcessLocalAndStartsEmpty(t *testing.T) {
	metrics := NewGrantMetrics(newCapturedLogger())

	if len(metrics.Snapshot()) != 0 {
		t.Errorf("bộ đếm mới phải rỗng, có = %v", metrics.Snapshot())
	}

	// A second instance stands in for a restarted process: it shares nothing with the
	// first, which is exactly why a restart loses the counts.
	metrics.RecordRejection(ReasonUnknownEventType, "user-1")
	restarted := NewGrantMetrics(newCapturedLogger())
	if len(restarted.Snapshot()) != 0 {
		t.Errorf("sau khi khởi động lại tiến trình, bộ đếm phải bắt đầu từ 0, có = %v", restarted.Snapshot())
	}
}
