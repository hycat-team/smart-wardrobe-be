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
	RoutingKeyFashionItemCreatedWardrobe = eventconstants.TopicFashionItemCreatedWardrobe
	RoutingKeyFashionItemCreatedBrand    = eventconstants.TopicFashionItemCreatedBrand
	RoutingKeyFashionProcessingExhausted = eventconstants.TopicFashionProcessingExhausted
	RoutingKeyWardrobeDeleted            = eventconstants.TopicWardrobeDeleted
	RoutingKeySubscriptionPlanChanged    = eventconstants.TopicSubscriptionPlanChanged
	RoutingKeyNotificationEmail          = eventconstants.TopicNotificationEmail + ".#"
	RoutingKeyAuditLog                   = eventconstants.TopicAuditLog
)
