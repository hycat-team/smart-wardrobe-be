package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/notification/application/dto"
	notificationerrors "smart-wardrobe-be/services/account/internal/modules/notification/domain/errors"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// silentLogger satisfies logger.Interface without emitting anything.
type silentLogger struct{}

func (silentLogger) Debug(string, ...zap.Field) {}
func (silentLogger) Info(string, ...zap.Field)  {}
func (silentLogger) Warn(string, ...zap.Field)  {}
func (silentLogger) Error(string, ...zap.Field) {}
func (silentLogger) Fatal(string, ...zap.Field) {}

// recordingEmailUC records every address it was asked to deliver to.
type recordingEmailUC struct {
	recipients []string
	err        error
}

func (r *recordingEmailUC) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	if r.err != nil {
		return r.err
	}
	r.recipients = append(r.recipients, to)
	return nil
}

// recordingPublisher captures what the worker defers to the retry queue.
type recordingPublisher struct {
	topics   []string
	payloads [][]byte
	err      error
}

func (p *recordingPublisher) PublishDirect(_ context.Context, topic string, payload []byte, _ map[string]any) error {
	if p.err != nil {
		return p.err
	}
	p.topics = append(p.topics, topic)
	p.payloads = append(p.payloads, payload)
	return nil
}

// memoryDedupe is an in-process stand-in for the Redis key set.
type memoryDedupe struct {
	keys map[string]bool
	err  error
}

func newMemoryDedupe() *memoryDedupe { return &memoryDedupe{keys: map[string]bool{}} }

func (d *memoryDedupe) Seen(_ context.Context, key string) (bool, error) {
	if d.err != nil {
		return false, d.err
	}
	return d.keys[key], nil
}

func (d *memoryDedupe) Mark(_ context.Context, key string, _ time.Duration) error {
	if d.err != nil {
		return d.err
	}
	d.keys[key] = true
	return nil
}

func emailWorker(emailEnabled bool, uc *recordingEmailUC) *EmailWorker {
	return emailWorkerWith(emailEnabled, uc, &recordingPublisher{}, newMemoryDedupe())
}

func emailWorkerWith(emailEnabled bool, uc *recordingEmailUC, publisher *recordingPublisher, dedupe *memoryDedupe) *EmailWorker {
	cfg := &config.Config{}
	cfg.Email.Enabled = emailEnabled
	// handle() is exercised directly, so the message client is never touched.
	return NewEmailWorker(nil, uc, publisher, dedupe, cfg, silentLogger{})
}

func payload(to string) dto.EmailEventPayload {
	return dto.EmailEventPayload{To: to, Subject: "subject", Body: "<p>body</p>", OutboxID: uuid.NewString()}
}

// bulkRunAccounts is the shape of the K2.2 + K2.5 bulk verification runs that
// exhausted the provider's daily allowance: 55 + 50 accounts, each of which
// leaves one pending outbox row behind.
const bulkRunAccounts = 125

// TestTransientFailureIsDeferredThenAcknowledged is the retry contract: a relay
// that is briefly unreachable must not cost the customer the email, and the
// message we are holding must be ACKed only after the retry copy is safely on the
// broker.
func TestTransientFailureIsDeferredThenAcknowledged(t *testing.T) {
	uc := &recordingEmailUC{err: notificationerrors.Transient(errors.New("dial tcp: connection refused"))}
	publisher := &recordingPublisher{}
	dedupe := newMemoryDedupe()
	w := emailWorkerWith(true, uc, publisher, dedupe)

	p := payload("khach@example.test")
	require.NoError(t, w.handle(context.Background(), p),
		"a deferred delivery must ACK the original message")

	require.Len(t, publisher.topics, 1)
	require.Equal(t, eventconstants.TopicNotificationEmailRetry, publisher.topics[0])

	var deferred dto.EmailEventPayload
	require.NoError(t, json.Unmarshal(publisher.payloads[0], &deferred))
	require.Equal(t, p.OutboxID, deferred.OutboxID, "the retry must carry the same dedupe key")
	require.Equal(t, 1, deferred.RetryCount)
	require.False(t, dedupe.keys[p.OutboxID],
		"a deferred delivery must NOT be marked as delivered, or the retry would skip itself")
}

// TestPermanentFailureIsDeadLettered keeps a bad address out of the retry loop:
// no amount of waiting fixes an invalid recipient.
func TestPermanentFailureIsDeadLettered(t *testing.T) {
	uc := &recordingEmailUC{err: notificationerrors.Permanent(errors.New("550 mailbox unavailable"))}
	publisher := &recordingPublisher{}
	w := emailWorkerWith(true, uc, publisher, newMemoryDedupe())

	err := w.handle(context.Background(), payload("khach@example.test"))

	var permanent event.PermanentError
	require.ErrorAs(t, err, &permanent)
	require.Empty(t, publisher.topics, "a permanent failure must not be deferred")
}

// TestTransientFailureAfterMaxAttemptsIsDeadLettered bounds the retry loop. Five
// minutes x four deferrals is already twenty minutes of the customer waiting; a
// fifth attempt would be a message nobody is watching anymore.
func TestTransientFailureAfterMaxAttemptsIsDeadLettered(t *testing.T) {
	uc := &recordingEmailUC{err: notificationerrors.Transient(errors.New("421 try again later"))}
	publisher := &recordingPublisher{}
	w := emailWorkerWith(true, uc, publisher, newMemoryDedupe())

	p := payload("khach@example.test")
	p.RetryCount = maxEmailDeliveryAttempts

	err := w.handle(context.Background(), p)

	var permanent event.PermanentError
	require.ErrorAs(t, err, &permanent)
	require.Empty(t, publisher.topics)
}

// TestDuplicateDeliveryIsSkipped is the at-least-once defence: a message
// redelivered after a lost ack must not reach the customer twice.
func TestDuplicateDeliveryIsSkipped(t *testing.T) {
	uc := &recordingEmailUC{}
	dedupe := newMemoryDedupe()
	p := payload("khach@example.test")
	require.NoError(t, dedupe.Mark(context.Background(), p.OutboxID, time.Hour))

	w := emailWorkerWith(true, uc, &recordingPublisher{}, dedupe)
	require.NoError(t, w.handle(context.Background(), p))

	require.Empty(t, uc.recipients, "the same outbox row must not be delivered twice")
}

// TestDedupeOutageFailsOpen: a Redis problem must not stop transactional email.
func TestDedupeOutageFailsOpen(t *testing.T) {
	uc := &recordingEmailUC{}
	dedupe := newMemoryDedupe()
	dedupe.err = errors.New("redis: connection refused")

	w := emailWorkerWith(true, uc, &recordingPublisher{}, dedupe)
	require.NoError(t, w.handle(context.Background(), payload("khach@example.test")))

	require.Len(t, uc.recipients, 1, "the email must still be delivered when dedupe is unavailable")
}

// TestPublisherFailureDeadLettersInsteadOfSpinning: the queue is classic, so a
// requeue would loop with no delivery limit. The DLQ is the recoverable outcome.
func TestPublisherFailureDeadLettersInsteadOfSpinning(t *testing.T) {
	uc := &recordingEmailUC{err: notificationerrors.Transient(errors.New("421 try again later"))}
	publisher := &recordingPublisher{err: errors.New("broker unreachable")}
	w := emailWorkerWith(true, uc, publisher, newMemoryDedupe())

	err := w.handle(context.Background(), payload("khach@example.test"))

	var permanent event.PermanentError
	require.ErrorAs(t, err, &permanent)
}

// TestDisabledWorkerDiscardsTheBacklogInsteadOfReleasingIt is the property 025
// T113 exists for.
//
// A bulk run writes ~125 pending outbox rows while the switch is off. The rows
// still reach the consumer — the outbox publisher is not disabled, and it must
// not be, because a row that stays unclaimed is a row that fires later. What
// decides the outcome is that the handler ACKNOWLEDGES each one instead of
// nacking or requeueing it: a requeued message stays on the broker and is
// delivered the instant the switch is turned back on.
//
// So the assertion is deliberately about the second run, not the first: after
// re-enabling, a single-account verification run must send exactly one email.
// 126 would mean the backlog was held, not discarded.
func TestDisabledWorkerDiscardsTheBacklogInsteadOfReleasingIt(t *testing.T) {
	uc := &recordingEmailUC{}
	disabled := emailWorker(false, uc)
	ctx := context.Background()

	for i := 0; i < bulkRunAccounts; i++ {
		// nil means handled: the handler acks, so the row is done with.
		require.NoError(t, disabled.handle(ctx, payload(fmt.Sprintf("bulk-%03d@example.test", i))),
			"a disabled handler must ACK, not fail — a failure would requeue the message")
	}
	require.Empty(t, uc.recipients, "nothing may be delivered while the switch is off")

	enabled := emailWorker(true, uc)
	require.NoError(t, enabled.handle(ctx, payload("single-run@example.test")))

	require.Equal(t, []string{"single-run@example.test"}, uc.recipients,
		"re-enabling must not release the backlog: one verification run sends exactly one email, not %d",
		bulkRunAccounts+1)
}

// TestEnabledWorkerDelivers is the control: with the switch on, the handler
// delivers every message it receives, so the test above cannot pass simply
// because the handler never delivers anything.
func TestEnabledWorkerDelivers(t *testing.T) {
	uc := &recordingEmailUC{}
	w := emailWorker(true, uc)
	ctx := context.Background()

	for _, to := range []string{"a@example.test", "b@example.test", "c@example.test"} {
		require.NoError(t, w.handle(ctx, payload(to)))
	}

	require.Equal(t, []string{"a@example.test", "b@example.test", "c@example.test"}, uc.recipients)
}

// TestDisabledWorkerDeliversNothingEvenForASingleMessage isolates the "no send"
// half of the property, without any bulk run in the picture.
func TestDisabledWorkerDeliversNothingEvenForASingleMessage(t *testing.T) {
	uc := &recordingEmailUC{}
	w := emailWorker(false, uc)

	require.NoError(t, w.handle(context.Background(), payload("single-run@example.test")))
	require.Empty(t, uc.recipients)
}
