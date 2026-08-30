package dto

// EmailPayload là payload JSON của sự kiện notification.email.* gửi qua queue.
// Trước đây là anonymous struct trong EmailWorker — tách thành DTO có tên
// để tái sử dụng và test được (FR-020).
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Channel string `json:"channel"`
}
