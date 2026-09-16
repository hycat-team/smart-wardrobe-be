package entities

func (AICostPolicy) TableName() string          { return "ai_cost_policies" }
func (AICostPolicyOperation) TableName() string { return "ai_cost_policy_operations" }
func (AIPolicyGrant) TableName() string         { return "ai_policy_grants" }
func (AIUsagePeriodLedger) TableName() string   { return "ai_usage_period_ledgers" }
func (AIUsageEvent) TableName() string          { return "ai_usage_events" }
func (AIModelConfig) TableName() string         { return "ai_model_configs" }
