package dto

// EmailEventPayload là payload JSON của sự kiện notification.email.* gửi qua queue,
// consumed by the notification email worker.
type EmailEventPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Channel string `json:"channel"`

	// OutboxID ties the delivery back to the shared.outbox_records row that
	// produced it. It is the dedupe key: a message redelivered after a lost ack
	// carries the same id, so the worker can tell "this is the same email" from
	// "this is another email to the same customer".
	OutboxID string `json:"outboxId,omitempty"`

	// RetryCount is how many times delivery has already been deferred to the delay
	// queue. It lives in the message rather than in the broker because a classic
	// queue has no delivery counter that survives a requeue.
	RetryCount int `json:"retryCount,omitempty"`
}
