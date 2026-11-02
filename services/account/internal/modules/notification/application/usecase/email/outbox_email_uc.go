package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	usecase_interfaces "smart-wardrobe-be/services/account/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/notification/application/templates"
	nchannel "smart-wardrobe-be/services/account/internal/modules/notification/domain/constants/channel"
	ntemplate "smart-wardrobe-be/services/account/internal/modules/notification/domain/constants/template"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"smart-wardrobe-be/shared/application/constants/eventconstants"

	"github.com/google/uuid"
)

// OutboxEmailUseCase composes transactional emails from embedded templates
// and writes them into the outbox within the caller's transaction (the outbox
// repository detects and reuses the tx from context). It is the application
// layer behind the notification contract (feature 017).
type OutboxEmailUseCase struct {
	outboxRepo        shared_repos.OutboxRepository
	tmplCache         map[ntemplate.Template]*template.Template
	userBriefContract identity_contract.IUserBriefContract
}

func NewOutboxEmailUseCase(
	outboxRepo shared_repos.OutboxRepository,
	userBriefContract identity_contract.IUserBriefContract,
) usecase_interfaces.IOutboxEmailUseCase {
	tmplCache := map[ntemplate.Template]*template.Template{
		ntemplate.Otp:                   template.Must(template.New("otp").Parse(templates.OtpTemplate)),
		ntemplate.ForgotPasswordOtp:     template.Must(template.New("forgot_password_otp").Parse(templates.ForgotPasswordTemplate)),
		ntemplate.PaymentSuccess:        template.Must(template.New("payment_success").Parse(templates.PaymentSuccessTemplate)),
		ntemplate.TopupSuccess:          template.Must(template.New("topup_success").Parse(templates.TopupSuccessTemplate)),
		ntemplate.RenewalSuccess:        template.Must(template.New("renewal_success").Parse(templates.RenewalSuccessTemplate)),
		ntemplate.SubscriptionExpired:   template.Must(template.New("subscription_expired").Parse(templates.SubscriptionExpiredTemplate)),
		ntemplate.SubscriptionExpiring:  template.Must(template.New("subscription_expiring").Parse(templates.SubscriptionExpiringTemplate)),
		ntemplate.WalletWarning:         template.Must(template.New("wallet_warning").Parse(templates.WalletWarningTemplate)),
		ntemplate.WalletPurchaseSuccess: template.Must(template.New("wallet_purchase_success").Parse(templates.WalletPurchaseSuccessTemplate)),
		ntemplate.AdminGrant:            template.Must(template.New("admin_grant").Parse(templates.AdminGrantTemplate)),
		// 025 T073: a campaign grant is a DIFFERENT promise from an administrator's
		// grant — the customer did nothing to earn it and must not be told they paid
		// — so it gets its own rendered template (R12).
		ntemplate.CampaignGrant: template.Must(template.New("campaign_grant").Parse(templates.CampaignGrantTemplate)),
	}
	return &OutboxEmailUseCase{
		outboxRepo:        outboxRepo,
		tmplCache:         tmplCache,
		userBriefContract: userBriefContract,
	}
}

// ProductName and BrandName are what the customer sees on every email. They live
// here rather than inside thirteen templates so a rename is one edit, and so two
// templates cannot end up disagreeing about what the product is called.
const (
	ProductName = "Closy"
	BrandName   = "HYCAT"
)

func (uc *OutboxEmailUseCase) Send(ctx context.Context, channel nchannel.Channel, tmplEnum ntemplate.Template, params map[string]any, to string) error {
	if channel != nchannel.Email {
		return fmt.Errorf("unsupported channel: %s", channel)
	}

	tmpl, ok := uc.tmplCache[tmplEnum]
	if !ok {
		return fmt.Errorf("unknown template: %s", tmplEnum)
	}

	subject, err := resolveSubject(tmplEnum)
	if err != nil {
		return err
	}

	// Copy rather than mutate: the caller's map belongs to the use case that
	// built it, and injecting identity into it would make a retry render with
	// keys it never set.
	renderParams := make(map[string]any, len(params)+2)
	for key, value := range params {
		renderParams[key] = value
	}
	renderParams["ProductName"] = ProductName
	renderParams["BrandName"] = BrandName

	var bodyBuf bytes.Buffer
	if err := tmpl.Execute(&bodyBuf, renderParams); err != nil {
		return fmt.Errorf("render template %s: %w", tmplEnum, err)
	}

	payload := map[string]any{
		"to":      to,
		"subject": subject,
		"body":    bodyBuf.String(),
		"channel": string(channel),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	record := &shared_repos.OutboxRecord{
		Topic:   eventconstants.TopicNotificationEmail,
		Payload: raw,
		Headers: []byte(`{}`),
		Status:  shared_repos.OutboxStatusPending,
	}
	return uc.outboxRepo.Create(ctx, record)
}

func (uc *OutboxEmailUseCase) SendByUserID(ctx context.Context, channel nchannel.Channel, tmplEnum ntemplate.Template, params map[string]any, userID uuid.UUID) error {
	user, err := uc.userBriefContract.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user for notification: %w", err)
	}
	if user == nil || user.Email == "" {
		return fmt.Errorf("user not found or has no email: %s", userID)
	}
	return uc.Send(ctx, channel, tmplEnum, params, user.Email)
}

// subjectPrefix is what every customer-facing subject carries. The product name
// lives in one constant, not inside each template, so a rename is one edit.
var subjectPrefix = "[" + ProductName + "] "

// subjectByTemplate is the whole subject vocabulary, WITHOUT the prefix.
//
// It is a map rather than a switch with a default: the default used to be
// "Thông báo", so a new template that nobody wired a subject for reached
// customers as "[Closy] Thông báo" and nothing failed. A missing entry is now an
// error at enqueue time, and TestEveryTemplateHasASubject walks template.All()
// so the gap fails in CI instead of in an inbox.
var subjectByTemplate = map[ntemplate.Template]string{
	ntemplate.Otp:                   "Mã xác thực OTP",
	ntemplate.ForgotPasswordOtp:     "Đặt lại mật khẩu",
	ntemplate.PaymentSuccess:        "Xác nhận thanh toán thành công",
	ntemplate.TopupSuccess:          "Nạp tiền vào ví thành công",
	ntemplate.RenewalSuccess:        "Gia hạn gói thành công",
	ntemplate.SubscriptionExpired:   "Gói dịch vụ của bạn đã hết hạn",
	ntemplate.SubscriptionExpiring:  "Gói dịch vụ của bạn sắp hết hạn",
	ntemplate.WalletWarning:         "Cảnh báo số dư ví không đủ",
	ntemplate.WalletPurchaseSuccess: "Mua gói thành công qua ví nội bộ",
	// 024 FR-083: the wording must not reveal whether other accounts were part of
	// the same administrator batch, and must not imply payment, because an
	// administrative grant is not a purchase.
	ntemplate.AdminGrant: "Gói của bạn đã được cấp thành công",
	// 025 FR-044/FR-045: says a gift was granted and nothing else. In particular
	// it does not say "promotion", "offer" or "discount" - a subject line
	// promising a benefit the customer did nothing to qualify for is what turns a
	// grant into an expectation. The prefix is added by resolveSubject, so it is
	// not repeated here.
	ntemplate.CampaignGrant: "Chúc Mừng Bạn Nhận Được Quà Tặng!",
}

// resolveSubject returns the full subject for a template, prefix included.
func resolveSubject(tmplEnum ntemplate.Template) (string, error) {
	subject, ok := subjectByTemplate[tmplEnum]
	if !ok {
		return "", fmt.Errorf("no subject configured for template %q", tmplEnum)
	}
	return subjectPrefix + subject, nil
}
