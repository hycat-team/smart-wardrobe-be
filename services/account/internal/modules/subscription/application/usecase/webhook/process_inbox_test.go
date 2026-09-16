package webhook

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/webhookprocessingstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

type captureInboxRepo struct {
	created []*entities.ProviderWebhookInbox
}

func (f *captureInboxRepo) Create(ctx context.Context, event *entities.ProviderWebhookInbox) error {
	f.created = append(f.created, event)
	return nil
}
func (f *captureInboxRepo) GetByHash(ctx context.Context, provider, hash string) (*entities.ProviderWebhookInbox, error) {
	return nil, nil
}
func (f *captureInboxRepo) GetByReference(ctx context.Context, provider, reference, eventCode string) (*entities.ProviderWebhookInbox, error) {
	return nil, nil
}
func (f *captureInboxRepo) MarkProcessed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return nil
}
func (f *captureInboxRepo) MarkRetry(ctx context.Context, id uuid.UUID, at time.Time, message string) error {
	return nil
}
func (f *captureInboxRepo) MarkInvestigation(ctx context.Context, id uuid.UUID, at time.Time, message string) error {
	return nil
}
func (f *captureInboxRepo) MarkFailed(ctx context.Context, id uuid.UUID, at time.Time, reason string) error {
	return nil
}
func (f *captureInboxRepo) Claim(ctx context.Context, limit int, lease time.Duration) ([]*entities.ProviderWebhookInbox, error) {
	return nil, nil
}

// allowedDBProcessingStatuses mirrors the provider_webhook_inbox_processing_status_check
// constraint so the unit suite fails first if code and schema drift apart again.
var allowedDBProcessingStatuses = map[string]struct{}{
	"received":               {},
	"processing":             {},
	"retry_required":         {},
	"processed":              {},
	"investigation_required": {},
	"failed":                 {},
}

func inboxTestPayload() *WebhookPayload {
	return &WebhookPayload{
		Code:    "00",
		Success: true,
		Data: WebhookPayloadData{
			OrderCode:     1789139571592729,
			Amount:        json.Number("5000"),
			Description:   "CLOSY1789139571592729",
			AccountNumber: "0353219821",
			Reference:     "FT26254717059531",
			Currency:      "VND",
			PaymentLinkId: "33cb7d2be9b14cc19015cdcd0ee9b622",
			Code:          "00",
			Desc:          "success",
		},
	}
}

func TestPersistWebhookInboxUsesLowercaseReceivedStatus(t *testing.T) {
	repo := &captureInboxRepo{}
	uc := &PaymentWebhookUseCase{inboxRepo: repo}

	inbox, err := uc.persistWebhookInbox(context.Background(), inboxTestPayload())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected exactly one inbox row created, got %d", len(repo.created))
	}
	got := repo.created[0].ProcessingStatus
	if got != string(webhookprocessingstatus.Received) {
		t.Fatalf("expected status %q, got %q", string(webhookprocessingstatus.Received), got)
	}
	if _, ok := allowedDBProcessingStatuses[got]; !ok {
		t.Fatalf("status %q violates provider_webhook_inbox_processing_status_check", got)
	}
	if inbox == nil || inbox.ProcessingStatus != got {
		t.Fatalf("expected returned inbox to carry status %q", got)
	}
}
