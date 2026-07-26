package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"smart-wardrobe-be/internal/modules/notification/infrastructure/communication/templates"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"
)

type notificationContractImpl struct {
	outboxRepo messaging.OutboxRepository
	tmplCache  map[TemplateEnum]*template.Template
}

func NewNotificationContract(outboxRepo messaging.OutboxRepository) INotificationContract {
	tmplCache := map[TemplateEnum]*template.Template{
		TemplateOtp:            template.Must(template.New("otp").Parse(templates.OtpTemplate)),
		TemplatePaymentSuccess: template.Must(template.New("payment_success").Parse(templates.PaymentSuccessTemplate)),
		TemplateTopupSuccess:   template.Must(template.New("topup_success").Parse(templates.TopupSuccessTemplate)),
		TemplateRenewalSuccess: template.Must(template.New("renewal_success").Parse(templates.RenewalSuccessTemplate)),
		TemplateRenewalFailed:  template.Must(template.New("renewal_failed").Parse(templates.RenewalFailedTemplate)),
		TemplateWalletWarning:  template.Must(template.New("wallet_warning").Parse(templates.WalletWarningTemplate)),
	}
	return &notificationContractImpl{
		outboxRepo: outboxRepo,
		tmplCache:  tmplCache,
	}
}

func (c *notificationContractImpl) Send(ctx context.Context, channel ChannelEnum, tmplEnum TemplateEnum, params map[string]any, to string) error {
	if channel != ChannelEmail {
		return fmt.Errorf("unsupported channel: %s", channel)
	}

	tmpl, ok := c.tmplCache[tmplEnum]
	if !ok {
		return fmt.Errorf("unknown template: %s", tmplEnum)
	}

	subject := c.resolveSubject(tmplEnum, params)

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

	record := &messaging.OutboxRecord{
		Topic:   eventconstants.TopicNotificationEmail,
		Payload: raw,
		Headers: json.RawMessage(`{}`),
		Status:  messaging.OutboxStatusPending,
	}

	if txDB := db.GetTx(ctx); txDB != nil {
		return c.outboxRepo.Create(db.InjectTx(ctx, txDB), record)
	}
	return c.outboxRepo.Create(ctx, record)
}

func (c *notificationContractImpl) resolveSubject(tmplEnum TemplateEnum, params map[string]any) string {
	switch tmplEnum {
	case TemplateOtp:
		return "Mã xác thực OTP"
	case TemplatePaymentSuccess:
		return "Xác nhận thanh toán thành công"
	case TemplateTopupSuccess:
		return "Nạp tiền vào ví thành công"
	case TemplateRenewalSuccess:
		return "Gia hạn gói thành công"
	case TemplateRenewalFailed:
		return "Gia hạn gói thất bại"
	case TemplateWalletWarning:
		return "Cảnh báo số dư ví không đủ"
	default:
		return "Thông báo từ SmartWardrobe"
	}
}
