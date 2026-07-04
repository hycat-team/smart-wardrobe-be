package entities

func (Brand) TableName() string                 { return "brands" }
func (BrandMember) TableName() string            { return "brand_members" }
func (BrandCustomer) TableName() string          { return "brand_customers" }
func (BrandBenefit) TableName() string           { return "brand_benefits" }
func (BenefitRedemption) TableName() string      { return "benefit_redemptions" }
func (BrandConversation) TableName() string      { return "brand_conversations" }
func (BrandConversationMessage) TableName() string { return "brand_conversation_messages" }
func (BrandItem) TableName() string              { return "brand_items" }
func (DigitalSampleResponse) TableName() string  { return "digital_sample_responses" }
func (LoyaltyProgram) TableName() string         { return "loyalty_programs" }
func (LoyaltyTier) TableName() string            { return "loyalty_tiers" }
func (LoyaltyAccount) TableName() string         { return "loyalty_accounts" }
func (LoyaltyPointTransaction) TableName() string { return "loyalty_point_transactions" }
func (LoyaltyPointLot) TableName() string        { return "loyalty_point_lots" }
func (BrandCustomerClaim) TableName() string     { return "brand_customer_claims" }
