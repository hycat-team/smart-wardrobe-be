package webhookprocessingstatus

type WebhookProcessingStatus string

const (
	Received              WebhookProcessingStatus = "received"
	Processing            WebhookProcessingStatus = "processing"
	Processed             WebhookProcessingStatus = "processed"
	RetryRequired         WebhookProcessingStatus = "retry_required"
	InvestigationRequired WebhookProcessingStatus = "investigation_required"
	Failed                WebhookProcessingStatus = "failed"
)
