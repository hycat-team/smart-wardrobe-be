package eventconstants

const (
	TopicWardrobeBatchUpload = "wardrobe.event.batch-upload"
	TopicWardrobeCreated     = "wardrobe.event.created"
	TopicWardrobeUpdated     = "wardrobe.event.updated"
	TopicWardrobeDeleted     = "wardrobe.event.deleted"

	TopicSubscriptionPlanChanged = "subscription.plan.changed"

	// Fashion Events
	TopicFashionInit                = "fashion.event.init"
	TopicFashionAnalyze             = "fashion.event.analyze"
	TopicFashionAnalyzed            = "fashion.event.analyzed"
	TopicFashionFailed              = "fashion.event.failed"
	TopicFashionItemCreated         = "fashion.item.created"
	TopicFashionAnalyzedWardrobe    = "fashion.event.analyzed.wardrobe"
	TopicFashionAnalyzedBrand       = "fashion.event.analyzed.brand"
	TopicFashionFailedWardrobe      = "fashion.event.failed.wardrobe"
	TopicFashionFailedBrand         = "fashion.event.failed.brand"
	TopicFashionProcessingWardrobe  = "fashion.event.processing.wardrobe"
	TopicFashionProcessingBrand     = "fashion.event.processing.brand"
	TopicFashionItemCreatedWardrobe = "fashion.item.created.wardrobe"
	TopicFashionItemCreatedBrand    = "fashion.item.created.brand"
	TopicFashionProcessingExhausted = "fashion.event.processing-exhausted"
	TopicNotificationEmail          = "notification.email"

	TopicAuditLog = "admin.audit.log"

	// Community events
	TopicOutfitDeleted     = "wardrobe.outfit.deleted"
	TopicUserStatusChanged = "identity.user.status.changed"

	// Identity events
	// TopicUserCreated is emitted by the account service for EVERY newly created
	// account (025, FR-005). It carries no personally identifiable data (FR-087):
	// a receiver that needs the email reads the account table instead.
	TopicUserCreated = "identity.user.created"

	ActionCreated = "created"
	ActionUpdated = "updated"
	ActionDeleted = "deleted"
)
