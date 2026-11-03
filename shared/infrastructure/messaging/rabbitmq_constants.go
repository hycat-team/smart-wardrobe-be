package messaging

import "smart-wardrobe-be/shared/application/constants/eventconstants"

// Exchange + routing key names là chi tiết vật lý của broker AMQP
// (chỉ topology/consumer implementation dùng); queue names nằm ở
// eventconstants — shared vocabulary mà worker cũng cần.
const (
	ExchangeName = "smart_wardrobe_exchange"
	ExchangeType = "topic"

	// Routing Keys
	RoutingKeyElasticsearchSyncWildcard  = "wardrobe.event.#"
	RoutingKeyFashionInit                = eventconstants.TopicFashionInit
	RoutingKeyFashionAnalyze             = eventconstants.TopicFashionAnalyze
	RoutingKeyFashionAnalyzed            = eventconstants.TopicFashionAnalyzed
	RoutingKeyFashionFailed              = eventconstants.TopicFashionFailed
	RoutingKeyFashionItemCreated         = eventconstants.TopicFashionItemCreated
	RoutingKeyFashionAnalyzedWardrobe    = eventconstants.TopicFashionAnalyzedWardrobe
	RoutingKeyFashionAnalyzedBrand       = eventconstants.TopicFashionAnalyzedBrand
	RoutingKeyFashionFailedWardrobe      = eventconstants.TopicFashionFailedWardrobe
	RoutingKeyFashionFailedBrand         = eventconstants.TopicFashionFailedBrand
	RoutingKeyFashionProcessingWardrobe  = eventconstants.TopicFashionProcessingWardrobe
	RoutingKeyFashionProcessingBrand     = eventconstants.TopicFashionProcessingBrand
	RoutingKeyFashionItemCreatedWardrobe = eventconstants.TopicFashionItemCreatedWardrobe
	RoutingKeyFashionItemCreatedBrand    = eventconstants.TopicFashionItemCreatedBrand
	RoutingKeyFashionProcessingExhausted = eventconstants.TopicFashionProcessingExhausted
	RoutingKeyWardrobeDeleted            = eventconstants.TopicWardrobeDeleted
	RoutingKeySubscriptionPlanChanged    = eventconstants.TopicSubscriptionPlanChanged
	RoutingKeyNotificationEmail          = eventconstants.TopicNotificationEmail + ".#"
	// RoutingKeyNotificationEmailRetry is a distinct word sequence on purpose: the
	// main queue's "notification.email.#" matches any multi-word suffix including
	// "notification.email.retry", so a retry published under that name would also
	// land on the main queue and be consumed immediately.
	RoutingKeyNotificationEmailRetry = eventconstants.TopicNotificationEmailRetry
	RoutingKeyAuditLog               = eventconstants.TopicAuditLog

	RoutingKeyOutfitDeleted     = eventconstants.TopicOutfitDeleted
	RoutingKeyUserStatusChanged = eventconstants.TopicUserStatusChanged
	// 033 T017: outfit-save signal for the style learning worker. References the
	// topic symbol so the outbox routing key and the binding cannot drift.
	RoutingKeyOutfitSaved = eventconstants.TopicOutfitSaved
	// 025 T003 / FR-093 (R7): references the topic SYMBOL, never a raw literal.
	// The topic also drives the outbox routing key, so two literals would be two
	// definitions of the same event that can silently drift apart.
	RoutingKeyUserCreated = eventconstants.TopicUserCreated
)
