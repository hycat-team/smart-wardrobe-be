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

// emailBody renders one template and returns the subject and body.
func emailBody(t *testing.T, uc *OutboxEmailUseCase, outbox *recordingOutbox, tmpl ntemplate.Template, params map[string]any, userID uuid.UUID) (string, string) {
	t.Helper()
	if err := uc.SendByUserID(t.Context(), nchannel.Email, tmpl, params, userID); err != nil {
		t.Fatalf("send %s email: %v", tmpl, err)
	}
	if len(outbox.records) == 0 {
		t.Fatalf("expected an outbox record for %s, got none", tmpl)
	}
	// The LAST record: a test may render several templates into one outbox, and
	// asserting a total here would make every such test depend on its sibling's
	// order.
	var payload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal(outbox.records[len(outbox.records)-1].Payload, &payload); err != nil {
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
	subject, body := emailBody(t, uc, outbox, ntemplate.CampaignGrant, map[string]any{
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
	_, body := emailBody(t, uc, outbox, ntemplate.CampaignGrant, map[string]any{
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
	_, campaignBody := emailBody(t, uc, outbox, ntemplate.CampaignGrant, map[string]any{
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

// TestSubscriptionExpiredEmailNamesTheLostPlanAndNeverLeaksInternals pins the
// reported production email.
//
// A customer whose Premium plan merely expired received "[Closy] Gia Hạn Gói Thất
// Bại", "Gói dịch vụ: Free Plan", "Lý do: downgraded_auto_renew_disabled". Three
// separate defects: nothing failed, the plan named was the one they were moved
// TO, and an internal enum was rendered as customer copy.
func TestSubscriptionExpiredEmailNamesTheLostPlanAndNeverLeaksInternals(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")
	subject, body := emailBody(t, uc, outbox, ntemplate.SubscriptionExpired, map[string]any{
		"PlanName":   "Premium Plan",
		"ExpiredAt":  "01/11/2026",
		"NeedsTopup": false,
	}, uuid.New())

	visible := visibleText(body)
	if !strings.Contains(visible, "Premium Plan") {
		t.Fatalf("the email must name the plan that ended, got:\n%s", visible)
	}
	if !strings.Contains(visible, "01/11/2026") {
		t.Fatalf("the email must state the expiry date, got:\n%s", visible)
	}
	// The customer did not fail at anything: the term ended and auto-renew was
	// off. "Thất bại" in the subject is what made a normal expiry read as an error.
	if strings.Contains(strings.ToLower(subject), "thất bại") {
		t.Fatalf("an expiring plan is not a failed renewal, subject = %q", subject)
	}
	if strings.Contains(strings.ToLower(visible), "thất bại") {
		t.Fatalf("the body must not call a normal expiry a failure, got:\n%s", visible)
	}
	// Internal enum values are operator-facing. The template receives a boolean
	// and picks Vietnamese copy; a raw code reaching a customer is the defect.
	for _, forbidden := range []string{"downgraded_", "auto_renew_disabled", "insufficient_balance", "_disabled"} {
		if strings.Contains(visible, forbidden) {
			t.Fatalf("the email must not render the internal reason code %q, got:\n%s", forbidden, visible)
		}
	}
	if strings.HasPrefix(subject, "[Closy] ") == false {
		t.Fatalf("expected the subject to carry the [Closy] prefix, got %q", subject)
	}
}

// TestSubscriptionExpiredEmailShowsReadableAmountsWhenTopupNeeded covers the
// wallet-shortfall branch: the customer's next action is a top-up, so the amount
// has to be readable. "1e+06" is not an amount a customer can act on.
func TestSubscriptionExpiredEmailShowsReadableAmountsWhenTopupNeeded(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")
	_, body := emailBody(t, uc, outbox, ntemplate.SubscriptionExpired, map[string]any{
		"PlanName":       "Premium Plan",
		"ExpiredAt":      "01/11/2026",
		"NeedsTopup":     true,
		"RequiredAmount": "1.000.000đ",
		"Balance":        "5.000đ",
	}, uuid.New())

	visible := visibleText(body)
	for _, want := range []string{"1.000.000đ", "5.000đ", "nạp thêm tiền"} {
		if !strings.Contains(visible, want) {
			t.Fatalf("expected %q in the shortfall email, got:\n%s", want, visible)
		}
	}
	// The auto-renew-off sentence must not appear here: the switch is ON, the
	// wallet is the obstacle.
	if strings.Contains(visible, "Bạn đang tắt tự động gia hạn") {
		t.Fatalf("the shortfall email must not blame the auto-renew switch, got:\n%s", visible)
	}
}

// TestForgotPasswordEmailIsNotTheRegistrationEmail pins a defect that was
// invisible from the outside: ForgotPasswordOtp.html was embedded but never
// registered, so every password-reset mail rendered the registration copy and
// told the customer "Bạn đang thực hiện thao tác đăng ký tài khoản".
func TestForgotPasswordEmailIsNotTheRegistrationEmail(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")
	params := map[string]any{"OTPCode": "123456", "ExpiryMinutes": 5}

	resetSubject, resetBody := emailBody(t, uc, outbox, ntemplate.ForgotPasswordOtp, params, uuid.New())
	visible := visibleText(resetBody)
	if !strings.Contains(strings.ToLower(visible), "đặt lại mật khẩu") {
		t.Fatalf("the reset email must say it is a password reset, got:\n%s", visible)
	}
	if strings.Contains(strings.ToLower(visible), "đăng ký tài khoản") {
		t.Fatalf("the reset email must not talk about registration, got:\n%s", visible)
	}
	if !strings.Contains(visible, "123456") {
		t.Fatalf("the reset email must carry the code, got:\n%s", visible)
	}

	// The registration mail and the reset mail are separate templates with
	// separate subjects; sharing either is how the wrong wording travels.
	regSubject, regBody := emailBody(t, uc, outbox, ntemplate.Otp, params, uuid.New())
	if resetSubject == regSubject {
		t.Fatalf("password reset and registration must not share a subject, both %q", resetSubject)
	}
	if visibleText(regBody) == visible {
		t.Fatalf("password reset and registration must not render the same body")
	}
	if !strings.Contains(strings.ToLower(visibleText(regBody)), "đăng ký tài khoản") {
		t.Fatalf("the registration email must still talk about registration, got:\n%s", visibleText(regBody))
	}
}

// TestPurchaseEmailsSayThePlanDoesNotAutoRenew closes the loop that produced the
// customer's complaint.
//
// No purchase turns auto-renew on - only the explicit PUT
// /subscriptions/me/auto-renew does - so a customer who bought a plan had no way
// to know it would simply stop. The expiry email then arrived as a surprise.
// CampaignGrant already carried this warning (FR-045); every other path that
// hands out a finite plan must say it too.
func TestPurchaseEmailsSayThePlanDoesNotAutoRenew(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")

	cases := []struct {
		name   string
		tmpl   ntemplate.Template
		params map[string]any
	}{
		{
			name: "payment success",
			tmpl: ntemplate.PaymentSuccess,
			params: map[string]any{
				"PlanName": "Premium Plan", "Amount": "5.000đ", "ExpiresAt": "01/11/2026",
			},
		},
		{
			name: "wallet purchase",
			tmpl: ntemplate.WalletPurchaseSuccess,
			params: map[string]any{
				"PlanName": "Premium Plan", "Amount": "5.000đ",
				"NewBalance": "95.000đ", "ExpiresAt": "01/11/2026",
			},
		},
		{
			name: "admin grant",
			tmpl: ntemplate.AdminGrant,
			params: map[string]any{
				"PlanName": "Premium Plan", "ExpiresAt": "01/11/2026",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, body := emailBody(t, uc, outbox, tc.tmpl, tc.params, uuid.New())
			visible := visibleText(body)
			if !strings.Contains(visible, "không tự động gia hạn") {
				t.Fatalf("%s must say the plan does not auto-renew, got:\n%s", tc.name, visible)
			}
			// The administrator path must not imply a payment (024 FR-083).
			if tc.tmpl == ntemplate.AdminGrant {
				for _, forbidden := range []string{"thanh toán thành công", "đã thanh toán", "hóa đơn"} {
					if strings.Contains(strings.ToLower(visible), forbidden) {
						t.Fatalf("an administrative grant must not imply payment, found %q in:\n%s", forbidden, visible)
					}
				}
				if strings.Contains(strings.ToLower(visible), "nâng cấp") {
					t.Fatalf("a grant may replace a higher plan, so the heading must not promise an upgrade, got:\n%s", visible)
				}
			}
		})
	}
}

// TestSubscriptionExpiringEmailTellsTheCustomerWhatWillHappen is the gap that
// made the expiry a surprise: the pre-expiry job skipped every account with
// auto-renew switched off, so those customers received no warning at all.
func TestSubscriptionExpiringEmailTellsTheCustomerWhatWillHappen(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")
	subject, body := emailBody(t, uc, outbox, ntemplate.SubscriptionExpiring, map[string]any{
		"PlanName":      "Premium Plan",
		"ExpiresAt":     "01/11/2026",
		"DaysRemaining": 3,
	}, uuid.New())

	visible := visibleText(body)
	for _, want := range []string{"Premium Plan", "01/11/2026", "gói Free", "tắt tự động gia hạn"} {
		if !strings.Contains(visible, want) {
			t.Fatalf("expected %q in the pre-expiry email, got:\n%s", want, visible)
		}
	}
	if strings.Contains(strings.ToLower(subject), "thất bại") {
		t.Fatalf("a pre-expiry reminder is not a failure, subject = %q", subject)
	}
}

// TestEveryTemplateRendersItsCallSiteParameters is the guard that would have
// caught the two production defects at the seam rather than in a customer's
// inbox: a template whose placeholders do not match the parameters its call site
// actually sends renders "<no value>" (or silently drops a line) instead of
// failing. The table below mirrors the params each use case passes - one row per
// send path - so a rename on either side breaks here.
func TestEveryTemplateRendersItsCallSiteParameters(t *testing.T) {
	uc, outbox := newCampaignEmailUseCase(t, "khach@example.com")

	// Mirrors the call sites: identity (register/reset), webhook top-up, webhook
	// payment, wallet purchase, renewal, expiry, pre-expiry, admin grant,
	// campaign grant.
	cases := []struct {
		name   string
		tmpl   ntemplate.Template
		params map[string]any
	}{
		{"registration otp", ntemplate.Otp, map[string]any{
			"OTPCode": "123456", "ExpiryMinutes": 5,
		}},
		{"forgot password otp", ntemplate.ForgotPasswordOtp, map[string]any{
			"OTPCode": "123456", "ExpiryMinutes": 5,
		}},
		{"topup success", ntemplate.TopupSuccess, map[string]any{
			"Amount": "1.000.000đ", "NewBalance": "1.500.000đ",
		}},
		{"payment success", ntemplate.PaymentSuccess, map[string]any{
			"PlanName": "Premium Plan", "Amount": "5.000đ", "ExpiresAt": "01/11/2026",
		}},
		{"wallet purchase success", ntemplate.WalletPurchaseSuccess, map[string]any{
			"PlanName": "Premium Plan", "Amount": "5.000đ",
			"NewBalance": "95.000đ", "ExpiresAt": "01/11/2026",
		}},
		{"renewal success", ntemplate.RenewalSuccess, map[string]any{
			"PlanName": "Premium Plan", "Amount": "5.000đ", "NewExpiresAt": "01/12/2026",
		}},
		{"subscription expired without topup", ntemplate.SubscriptionExpired, map[string]any{
			"PlanName": "Premium Plan", "ExpiredAt": "01/11/2026", "NeedsTopup": false,
		}},
		{"subscription expired with topup", ntemplate.SubscriptionExpired, map[string]any{
			"PlanName": "Premium Plan", "ExpiredAt": "01/11/2026", "NeedsTopup": true,
			"RequiredAmount": "1.000.000đ", "Balance": "5.000đ",
		}},
		{"subscription expiring", ntemplate.SubscriptionExpiring, map[string]any{
			"PlanName": "Premium Plan", "ExpiresAt": "01/11/2026", "DaysRemaining": 3,
		}},
		{"wallet warning", ntemplate.WalletWarning, map[string]any{
			"PlanName": "Premium Plan", "ExpiresAt": "01/11/2026", "DaysRemaining": 3,
			"Balance": "5.000đ", "RequiredAmount": "1.000.000đ",
		}},
		{"admin grant", ntemplate.AdminGrant, map[string]any{
			"PlanName": "Premium Plan", "ExpiresAt": "01/11/2026",
		}},
		{"campaign grant", ntemplate.CampaignGrant, map[string]any{
			"PlanName": "Premium Plan", "DurationText": "30 ngày",
			"GrantedOn": "01/10/2026", "ExpiresAt": "01/11/2026",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, body := emailBody(t, uc, outbox, tc.tmpl, tc.params, uuid.New())
			if strings.Contains(body, "<no value>") {
				t.Fatalf("%s renders an unfilled placeholder; its params and its template disagree:\n%s", tc.name, body)
			}
			// Every parameter the call site sends must be the one the template
			// reads, so a readable value has to survive into the body.
			for key, value := range tc.params {
				text, ok := value.(string)
				if !ok || text == "" {
					continue
				}
				if !strings.Contains(visibleText(body), text) {
					t.Fatalf("%s does not render param %q (%q); the template may be reading a different key:\n%s",
						tc.name, key, text, visibleText(body))
				}
			}
		})
	}
}
