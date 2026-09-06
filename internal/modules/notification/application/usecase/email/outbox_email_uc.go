package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	usecase_interfaces "smart-wardrobe-be/internal/modules/notification/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/notification/application/templates"
	nchannel "smart-wardrobe-be/internal/modules/notification/domain/constants/channel"
	ntemplate "smart-wardrobe-be/internal/modules/notification/domain/constants/template"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"

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
		Headers: json.RawMessage(`{}`),
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
	switch tmplEnum {
	case ntemplate.Otp:
		return "Mã xác thực OTP"
	case ntemplate.PaymentSuccess:
		return "Xác nhận thanh toán thành công"
	case ntemplate.TopupSuccess:
		return "Nạp tiền vào ví thành công"
	case ntemplate.RenewalSuccess:
		return "Gia hạn gói thành công"
	case ntemplate.RenewalFailed:
		return "Gia hạn gói thất bại"
	case ntemplate.WalletWarning:
		return "Cảnh báo số dư ví không đủ"
	case ntemplate.WalletPurchaseSuccess:
		return "Mua gói thành công qua ví nội bộ"
	default:
		return "Thông báo từ SmartWardrobe"
	}
}
