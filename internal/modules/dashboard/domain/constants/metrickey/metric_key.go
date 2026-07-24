package metrickey

type MetricKey string

const (
	TotalUsers                MetricKey = "total_users"
	ActiveUsersDau            MetricKey = "active_users_dau"
	TotalBrands               MetricKey = "total_brands"
	PendingReviewBrands       MetricKey = "pending_review_brands"
	AIRequestsCount           MetricKey = "ai_requests_count"
	SubscriptionRevenueVND    MetricKey = "subscription_revenue_vnd"
	NetAIMarginVND            MetricKey = "net_ai_margin_vnd"
	TotalMembers              MetricKey = "total_members"
	PointsIssued              MetricKey = "points_issued"
	PointsRedeemed            MetricKey = "points_redeemed"
	ActiveVouchers            MetricKey = "active_vouchers"
	TotalWardrobeItems        MetricKey = "total_wardrobe_items"
	TotalWardrobeValueVND     MetricKey = "total_wardrobe_value_vnd"
	VouchersRedeemed          MetricKey = "vouchers_redeemed"
	DigitalSampleVotesCount   MetricKey = "digital_sample_votes_count"
	DigitalSampleAvgRating    MetricKey = "digital_sample_avg_rating"
	OpenCSTickets             MetricKey = "open_cs_tickets"
	CSFirstResponseAvgSeconds MetricKey = "cs_first_response_avg_seconds"
	PointsExpiring30d         MetricKey = "points_expiring_30d"
	PointsLiabilityValueVND   MetricKey = "points_liability_value_vnd"

	// Admin expansion metrics
	NewUsers     MetricKey = "new_users"
	AIErrorCount MetricKey = "ai_error_count"
	AIErrorRate  MetricKey = "ai_error_rate"

	// Brand expansion metrics
	NewCustomers           MetricKey = "new_customers"
	CustomerClaimRate      MetricKey = "customer_claim_rate"
	BenefitRedemptionCount MetricKey = "benefit_redemption_count"
	BrandItemCount         MetricKey = "brand_item_count"
	BrandItemCountByType   MetricKey = "brand_item_count_by_type"
)
