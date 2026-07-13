package eventconstants

const (
	TopicWardrobeBatchUpload = "wardrobe.event.batch-upload"
	TopicWardrobeCreated     = "wardrobe.event.created"
	TopicWardrobeUpdated     = "wardrobe.event.updated"
	TopicWardrobeDeleted     = "wardrobe.event.deleted"

	TopicSubscriptionPlanChanged = "subscription.plan.changed"

	// Fashion Events
	TopicFashionInit    = "fashion.event.init"
	TopicFashionAnalyze = "fashion.event.analyze"

	ActionCreated = "created"
	ActionUpdated = "updated"
	ActionDeleted = "deleted"
)
