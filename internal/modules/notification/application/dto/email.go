package dto

// EmailEventPayload là payload JSON của sự kiện notification.email.* gửi qua queue,
// consumed by the notification email worker.
type EmailEventPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Channel string `json:"channel"`
}
