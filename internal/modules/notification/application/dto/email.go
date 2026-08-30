package dto

// EmailPayload là payload JSON của sự kiện notification.email.* gửi qua queue.
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Channel string `json:"channel"`
}
