package entities

// TableName returns table name mapping for SubscriptionPlan
func (SubscriptionPlan) TableName() string {
	return "subscription_plans"
}

// TableName returns table name mapping for User
func (User) TableName() string {
	return "users"
}

// TableName returns table name mapping for UserSubscription
func (UserSubscription) TableName() string {
	return "user_subscriptions"
}

// TableName returns table name mapping for UserDailyQuota
func (UserDailyQuota) TableName() string {
	return "user_daily_quotas"
}

// TableName returns table name mapping for UserStyleProfile
func (UserStyleProfile) TableName() string {
	return "user_style_profiles"
}

// TableName returns table name mapping for RefreshToken
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// TableName returns table name mapping for PaymentHistory
func (PaymentHistory) TableName() string {
	return "payment_histories"
}

// TableName returns table name mapping for ConversationalContext
func (ConversationalContext) TableName() string {
	return "conversational_contexts"
}

// TableName returns table name mapping for Message
func (Message) TableName() string {
	return "messages"
}

// TableName returns table name mapping for Category
func (Category) TableName() string {
	return "categories"
}

// TableName returns table name mapping for WardrobeItem
func (WardrobeItem) TableName() string {
	return "wardrobe_items"
}

// TableName returns table name mapping for Outfit
func (Outfit) TableName() string {
	return "outfits"
}

// TableName returns table name mapping for OutfitItem
func (OutfitItem) TableName() string {
	return "outfit_items"
}

// TableName returns table name mapping for Post
func (Post) TableName() string {
	return "posts"
}

// TableName returns table name mapping for PostScoreSnapshot
func (PostScoreSnapshot) TableName() string {
	return "post_score_snapshots"
}

// TableName returns table name mapping for PostItem
func (PostItem) TableName() string {
	return "post_items"
}

// TableName returns table name mapping for Comment
func (Comment) TableName() string {
	return "comments"
}

// TableName returns table name mapping for Like
func (Like) TableName() string {
	return "likes"
}
