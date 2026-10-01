package email

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/dto/identity"
	nchannel "smart-wardrobe-be/services/account/internal/modules/notification/domain/constants/channel"
	ntemplate "smart-wardrobe-be/services/account/internal/modules/notification/domain/constants/template"
	sharedrepos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

// 025 T068-T069: what the customer is told after a campaign grant.
//
// The content assertions live here rather than in the campaign use case because
// the TEMPLATE is the deliverable: the use case test proves one email is enqueued
// with the campaign template, and this proves what that template says once
// rendered. Splitting them means a wording change fails here, where the wording
// lives, and a wiring change fails there.

// recordingOutbox captures the outbox rows the use case writes.
type recordingOutbox struct {
	mu      sync.Mutex
	records []*sharedrepos.OutboxRecord
}

func (r *recordingOutbox) Create(ctx context.Context, record *sharedrepos.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
	return nil
}

// The publisher half of the outbox is not exercised by these tests — they are
// about what the email says, not about how it reaches RabbitMQ. Implemented as
// inert methods rather than a narrower interface so the stub stays a drop-in for
// the real repository.
func (r *recordingOutbox) ClaimToPublish(ctx context.Context, limit int, maxAttempts int, retryAfter time.Duration) ([]sharedrepos.OutboxRecord, error) {
	return nil, nil
}

func (r *recordingOutbox) MarkPublished(ctx context.Context, id uuid.UUID) error { return nil }

func (r *recordingOutbox) MarkFailed(ctx context.Context, id uuid.UUID, err error) error { return nil }

func (r *recordingOutbox) DeletePublishedBefore(ctx context.Context, since time.Time) (int64, error) {
	return 0, nil
}

func (r *recordingOutbox) CountPending(ctx context.Context) (int64, error) { return 0, nil }

func (r *recordingOutbox) SendNotification(ctx context.Context) error { return nil }

// stubUserBrief returns one account with an address, which is all `SendByUserID`
// needs (FR-047 covers the account with none).
type stubUserBrief struct {
	email string
}

func (s stubUserBrief) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefContractDTO, error) {
	return &dto.UserBriefContractDTO{ID: userID, Email: s.email}, nil
}

func (s stubUserBrief) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefContractDTO, error) {
	return nil, nil
}

func (s stubUserBrief) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error) {
	return nil, nil
}

func (s stubUserBrief) SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*dto.UserBriefContractDTO, error) {
	return nil, nil
}

// visibleText strips the markup so an assertion is about what the customer READS.
//
// The template's CSS carries numbers of its own (`600px`, `15px`), and a check
// over the raw HTML would either trip on those or — worse — be loosened until it
// stopped tripping on them, which is how a real giveaway count slips through.
func visibleText(body string) string {
	var b strings.Builder
	depth := 0
	for _, r := range body {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func newCampaignEmailUseCase(t *testing.T, email string) (*OutboxEmailUseCase, *recordingOutbox) {
	t.Helper()
	outbox := &recordingOutbox{}
	uc := NewOutboxEmailUseCase(outbox, stubUserBrief{email: email})
	return uc.(*OutboxEmailUseCase), outbox
}

// emailBody renders the campaign template and returns the subject and body.
func emailBody(t *testing.T, uc *OutboxEmailUseCase, outbox *recordingOutbox, params map[string]any, userID uuid.UUID) (string, string) {
	t.Helper()
	if err := uc.SendByUserID(t.Context(), nchannel.Email, ntemplate.CampaignGrant, params, userID); err != nil {
		t.Fatalf("send campaign email: %v", err)
	}
	if len(outbox.records) != 1 {
		t.Fatalf("expected exactly 1 outbox record, got %d", len(outbox.records))
	}
	var payload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal(outbox.records[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}
	return payload.Subject, payload.Body
}

// TestCampaignGrantEmailStatesPlanAndExpiryAndNoAutoRenew is T068 / SC-005.
//
// The four things FR-044 and FR-045 require, asserted on the RENDERED output:
// the plan's name, the date it ends, that it does not renew by itself, and no
// suggestion that anything was paid.
func TestCampaignGrantEmailStatesPlanAndExpiryAndNoAutoRenew(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")
	subject, body := emailBody(t, uc, outbox, map[string]any{
		"PlanName":  "Premium Tháng",
		"ExpiresAt": "01/11/2026",
	}, uuid.New())

	if !strings.Contains(body, "Premium Tháng") {
		t.Fatalf("expected the email to name the plan, got:\n%s", body)
	}
	if !strings.Contains(body, "01/11/2026") {
		t.Fatalf("expected the email to state the expiry date, got:\n%s", body)
	}
	// FR-045: the customer must know the plan will NOT charge them again. Without
	// this line a premium plan arriving by email reads as a subscription they
	// somehow signed up for.
	if !strings.Contains(body, "không tự động gia hạn") {
		t.Fatalf("expected the email to say the plan does not auto-renew (FR-045), got:\n%s", body)
	}
	// FR-045 again, from the other side: nothing may imply a payment happened.
	for _, forbidden := range []string{"thanh toán thành công", "đã thanh toán", "giao dịch", "hóa đơn", "hoàn tiền"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("the email must not suggest a payment was made (FR-045), found %q in:\n%s", forbidden, body)
		}
	}
	if !strings.HasPrefix(subject, "[Closy] ") {
		t.Fatalf("expected the subject to carry the [Closy] prefix, got %q", subject)
	}
}

// TestCampaignGrantEmailDoesNotMentionCampaignQuota is FR-046: the remaining
// giveaway count is irrelevant to this customer and is how a campaign becomes a
// queue at the door.
func TestCampaignGrantEmailDoesNotMentionCampaignQuota(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")
	_, body := emailBody(t, uc, outbox, map[string]any{
		"PlanName":  "Premium Tháng",
		"ExpiresAt": "01/11/2026",
	}, uuid.New())
	visible := visibleText(body)

	// Vietnamese words for the campaign accounting, plus the digits that would give
	// it away.
	for _, forbidden := range []string{"chiến dịch", "chương trình", "suất", "còn lại", "số lượng", "hạn mức", "ưu đãi", "promo"} {
		if strings.Contains(strings.ToLower(visible), forbidden) {
			t.Fatalf("the email must not mention campaign accounting (FR-046), found %q in:\n%s", forbidden, visible)
		}
	}
	// A giveaway count would appear as a bare number in the prose. The date is
	// removed first, and the MARKUP is stripped before the check, because the
	// template's own CSS is full of three-digit values (`600px`, `15px`) that have
	// nothing to do with the campaign.
	prose := strings.ReplaceAll(visible, "01/11/2026", "")
	for _, field := range strings.FieldsFunc(prose, func(r rune) bool { return r < '0' || r > '9' }) {
		if len(field) >= 2 {
			t.Fatalf("the email must not carry a giveaway count (FR-046), found the number %q in:\n%s", field, visible)
		}
	}
}

// TestCampaignGrantEmailIsNotSentWithoutAnAddress is FR-047: an account with no
// address is still granted the plan; the email is skipped.
func TestCampaignGrantEmailIsNotSentWithoutAnAddress(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "")

	err := uc.SendByUserID(t.Context(), nchannel.Email, ntemplate.CampaignGrant, map[string]any{
		"PlanName":  "Premium Tháng",
		"ExpiresAt": "01/11/2026",
	}, uuid.New())
	if err == nil {
		t.Fatalf("expected an error when the account has no address, so the caller can log it (FR-047)")
	}
	if got := len(outbox.records); got != 0 {
		t.Fatalf("expected NO outbox record for an account with no address, got %d", got)
	}
}

// TestCampaignGrantEmailSubjectIsDistinctFromAdminGrant guards R12: the two
// audiences must not be able to read each other's promise.
//
// If the subjects converged, a campaign customer would be reading text authored
// for an administrator, and any later edit to the manual wording would silently
// change what a campaign customer is told.
func TestCampaignGrantEmailSubjectIsDistinctFromAdminGrant(t *testing.T) {
	campaignSubject := resolveSubject(ntemplate.CampaignGrant)
	adminSubject := resolveSubject(ntemplate.AdminGrant)
	if campaignSubject == adminSubject {
		t.Fatalf("campaign and administrator grant emails must not share a subject (R12)")
	}
	for _, subject := range []string{campaignSubject, adminSubject} {
		if !strings.HasPrefix(subject, "[Closy] ") {
			t.Fatalf("expected every subject to carry the [Closy] prefix, got %q", subject)
		}
	}
}

// TestCampaignGrantEmailUsesItsOwnTemplate proves the campaign path does not
// render the administrator's HTML.
//
// A reuse would be invisible in production — both templates render, both say a
// plan was granted — and would show up only as a campaign customer receiving
// wording written for someone an operator acted on.
func TestCampaignGrantEmailUsesItsOwnTemplate(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")
	_, campaignBody := emailBody(t, uc, outbox, map[string]any{
		"PlanName":  "Premium Tháng",
		"ExpiresAt": "01/11/2026",
	}, uuid.New())

	if err := uc.Send(t.Context(), nchannel.Email, ntemplate.AdminGrant, map[string]any{
		"PlanName":  "Premium Tháng",
		"ExpiresAt": "01/11/2026",
	}, "khach@example.com"); err != nil {
		t.Fatalf("send admin grant email: %v", err)
	}
	var adminPayload struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(outbox.records[1].Payload, &adminPayload); err != nil {
		t.Fatalf("unmarshal admin outbox payload: %v", err)
	}
	if campaignBody == adminPayload.Body {
		t.Fatalf("campaign and administrator grant emails must render DIFFERENT templates (R12)")
	}
}
