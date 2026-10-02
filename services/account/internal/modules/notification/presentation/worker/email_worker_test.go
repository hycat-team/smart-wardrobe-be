package worker

import (
	"context"
	"fmt"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/notification/application/dto"
	"smart-wardrobe-be/shared/config"

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
}

func (r *recordingEmailUC) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	r.recipients = append(r.recipients, to)
	return nil
}

func emailWorker(emailEnabled bool, uc *recordingEmailUC) *EmailWorker {
	cfg := &config.Config{}
	cfg.Email.Enabled = emailEnabled
	// handle() is exercised directly, so the message client is never touched.
	return NewEmailWorker(nil, uc, cfg, silentLogger{})
}

func payload(to string) dto.EmailEventPayload {
	return dto.EmailEventPayload{To: to, Subject: "subject", Body: "<p>body</p>"}
}

// bulkRunAccounts is the shape of the K2.2 + K2.5 bulk verification runs that
// exhausted the provider's daily allowance: 55 + 50 accounts, each of which
// leaves one pending outbox row behind.
const bulkRunAccounts = 125

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
