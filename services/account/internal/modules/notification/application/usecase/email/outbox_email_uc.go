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
		ntemplate.PaymentSuccess:        template.Must(template.New("payment_success").Parse(templates.PaymentSuccessTemplate)),
		ntemplate.TopupSuccess:          template.Must(template.New("topup_success").Parse(templates.TopupSuccessTemplate)),
		ntemplate.RenewalSuccess:        template.Must(template.New("renewal_success").Parse(templates.RenewalSuccessTemplate)),
		ntemplate.RenewalFailed:         template.Must(template.New("renewal_failed").Parse(templates.RenewalFailedTemplate)),
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

func (uc *OutboxEmailUseCase) Send(ctx context.Context, channel nchannel.Channel, tmplEnum ntemplate.Template, params map[string]any, to string) error {
	if channel != nchannel.Email {
		return fmt.Errorf("unsupported channel: %s", channel)
	}

	tmpl, ok := uc.tmplCache[tmplEnum]
	if !ok {
		return fmt.Errorf("unknown template: %s", tmplEnum)
	}

	subject := resolveSubject(tmplEnum)

	var bodyBuf bytes.Buffer
	if err := tmpl.Execute(&bodyBuf, params); err != nil {
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

func resolveSubject(tmplEnum ntemplate.Template) string {
	const prefix = "[Closy] "
	var subject string
	switch tmplEnum {
	case ntemplate.Otp:
		subject = "Mã xác thực OTP"
	case ntemplate.PaymentSuccess:
		subject = "Xác nhận thanh toán thành công"
	case ntemplate.TopupSuccess:
		subject = "Nạp tiền vào ví thành công"
	case ntemplate.RenewalSuccess:
		subject = "Gia hạn gói thành công"
	case ntemplate.RenewalFailed:
		subject = "Gia hạn gói thất bại"
	case ntemplate.WalletWarning:
		subject = "Cảnh báo số dư ví không đủ"
	case ntemplate.WalletPurchaseSuccess:
		subject = "Mua gói thành công qua ví nội bộ"
	case ntemplate.AdminGrant:
		// 024 FR-083: the wording must not reveal whether other accounts were
		// part of the same administrator batch, and must not imply payment,
		// because an administrative grant is not a purchase.
		subject = "Gói của bạn đã được cấp thành công"
	case ntemplate.CampaignGrant:
		// 025 FR-044/FR-045: says the plan was granted and nothing else. In
		// particular it does not say "promotion", "offer" or "gift" — a subject
		// line promising a benefit the customer did nothing to qualify for is
		// what turns a grant into an expectation.
		subject = "Bạn đã được tặng gói hội viên"
	default:
		subject = "Thông báo"
	}
	return prefix + subject
}
