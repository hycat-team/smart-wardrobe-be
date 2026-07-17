# Thiết kế Mô hình Dữ liệu: Chuẩn hoá các Hằng số Kỹ thuật của AI Executor

## Các Thực thể Thay đổi (Entity Modifications)

### 1. Thực thể `AIModelConfig`
Nằm tại [model_config.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/entities/model_config.go) (Đổi tên từ `pricing.go`)

```go
type AIModelConfig struct {
	AuditableEntity
	Provider            string          `gorm:"type:varchar(50);not null"`
	Model               string          `gorm:"type:varchar(100);not null"`
	InputUSDPerMillion  decimal.Decimal `gorm:"type:numeric(20,8);not null"`
	OutputUSDPerMillion decimal.Decimal `gorm:"type:numeric(20,8);not null"`
	Currency            string          `gorm:"type:varchar(10);not null;default:'VND'"`
	USDToVND            decimal.Decimal `gorm:"type:numeric(20,4);not null"`
	Version             string          `gorm:"type:varchar(50);not null;default:'v1'"`
	EffectiveFrom       time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	EffectiveTo         *time.Time      `gorm:"type:timestamptz"`
	CreatedBy           *uuid.UUID      `gorm:"type:uuid"`
}
```

### 2. Thực thể `AICostPolicyOperation`
Nằm tại [policy.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/entities/policy.go)

```go
type AICostPolicyOperation struct {
	AuditableEntity
	PolicyID               uuid.UUID               `gorm:"type:uuid;not null"`
	Operation              aioperation.AIOperation `gorm:"type:varchar(30);not null"`
	NormalRoute            airoute.AIRoute         `gorm:"type:varchar(50);not null"`
	ReducedRoute           airoute.AIRoute         `gorm:"type:varchar(50);not null"`
	FreeRoute              airoute.AIRoute         `gorm:"type:varchar(50);not null"`
	NormalMaxInputTokens   int                     `gorm:"type:int;not null"`
	NormalMaxOutputTokens  int                     `gorm:"type:int;not null"`
	ReducedMaxInputTokens  int                     `gorm:"type:int;not null"`
	ReducedMaxOutputTokens int                     `gorm:"type:int;not null"`
	MaxPaidAttemptsPerDay  int                     `gorm:"type:int;not null"`
	PaidFallbackEnabled    bool                    `gorm:"type:boolean;not null"`
	IsEnabled              bool                    `gorm:"type:boolean;not null"`
}
```

### 3. Thực thể `AIUsageEvent`
Nằm tại [usage.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/domain/entities/usage.go)

```go
type AIUsageEvent struct {
	AuditableEntity
	RequestID                uuid.UUID                             `gorm:"type:uuid;not null;uniqueIndex"`
	LedgerID                 *uuid.UUID                            `gorm:"type:uuid"`
	UserID                   *uuid.UUID                            `gorm:"type:uuid"`
	PricingConfigID          *uuid.UUID                            `gorm:"type:uuid;column:pricing_config_id"`
	Operation                aioperation.AIOperation               `gorm:"type:varchar(30);not null"`
	LogicalRoute             airoute.AIRoute                       `gorm:"type:varchar(50);not null"`
	Provider                 *string                               `gorm:"type:varchar(50)"`
	Model                    *string                               `gorm:"type:varchar(150)"`
	InputUSDPerMillion       *decimal.Decimal                      `gorm:"type:numeric(18,8)"`
	OutputUSDPerMillion      *decimal.Decimal                      `gorm:"type:numeric(18,8)"`
	USDToVND                 *decimal.Decimal                      `gorm:"type:numeric(18,4)"`
	PromptTokens             int64                                 `gorm:"type:bigint;not null"`
	OutputTokens             int64                                 `gorm:"type:bigint;not null"`
	ThinkingTokens           int64                                 `gorm:"type:bigint;not null"`
	ReservedCostMicroVND     int64                                 `gorm:"type:bigint;not null"`
	ActualCostMicroVND       int64                                 `gorm:"type:bigint;not null"`
	EstimatedMaxCostMicroVND int64                                 `gorm:"type:bigint;not null"`
	Status                   aiusageeventstatus.AIUsageEventStatus `gorm:"type:varchar(30);not null"`
	FinishReason             *string                               `gorm:"type:varchar(100)"`
	ErrorCode                *string                               `gorm:"type:varchar(100)"`
	EstimatedPromptTokens    *int64                                `gorm:"type:bigint"`
	TokenEstimationMethod    *tokenestimationmethod.Method         `gorm:"type:varchar(30)"`
	TokenCountLatencyMs      *int64                                `gorm:"type:bigint"`
	SentAt                   *time.Time                            `gorm:"type:timestamp with time zone"`
	CompletedAt              *time.Time                            `gorm:"type:timestamp with time zone"`
	UnknownExpiresAt         *time.Time                            `gorm:"type:timestamp with time zone"`
}
```

## Các DTO Thay đổi (DTO Modifications)

### 1. DTO `ExecutionPlan`
Nằm tại [cost.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/dto/cost.go)

```go
type ExecutionPlan struct {
	RequestID             uuid.UUID
	UserID                uuid.UUID
	Operation             aioperation.AIOperation
	Provider              string
	Model                 string
	Route                 airoute.AIRoute
	MaxInputTokens        int
	MaxOutputTokens       int
	EstimatedPromptTokens int64
}
```

### 2. DTO `AICostDecision`
Nằm tại [cost.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/dto/cost.go)

```go
type AICostDecision struct {
	RequestID       uuid.UUID
	Route           airoute.AIRoute
	MaxInputTokens  int
	MaxOutputTokens int
	Tracked         bool
}
```

### 3. DTO `AIModelConfigDTO` (Đổi tên từ `PricingConfigDTO`)
Nằm tại [model_config.go](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/internal/modules/aiexecutor/application/dto/model_config.go) (Đổi tên từ `pricing.go`)

```go
type AIModelConfigDTO struct {
	ID                  uuid.UUID       `json:"id"`
	Provider            string          `json:"provider"`
	Model               string          `json:"model"`
	InputUSDPerMillion  decimal.Decimal `json:"input_usd_per_million"`
	OutputUSDPerMillion decimal.Decimal `json:"output_usd_per_million"`
	USDToVND            decimal.Decimal `json:"usd_to_vnd"`
}
```

## Cấu trúc Migration Hợp nhất (Consolidated Migration)
Lưu tại [20260704062159_init_ai_executor_module.sql](file:///c:/0LamViec/0FPTU/7-semester-7/exe101/projects/smart-wardrobe-be/database/migrations/20260704062159_init_ai_executor_module.sql)

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE ai_model_configs (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider                 VARCHAR(50)  NOT NULL,
    model                    VARCHAR(100) NOT NULL,
    input_usd_per_million    NUMERIC(20,8) NOT NULL,
    output_usd_per_million   NUMERIC(20,8) NOT NULL,
    currency                 VARCHAR(10)  NOT NULL DEFAULT 'VND',
    usd_to_vnd               NUMERIC(20,4) NOT NULL,
    version                  VARCHAR(50)  NOT NULL DEFAULT 'v1',
    effective_from           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    effective_to             TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         REFERENCES users(id),
    CONSTRAINT uq_provider_model_active UNIQUE (provider, model, effective_from)
);

ALTER TABLE user_ai_policy_grants RENAME TO ai_policy_grants;

ALTER TABLE ai_usage_events
    ADD COLUMN pricing_config_id UUID REFERENCES ai_model_configs(id),
    DROP COLUMN pricing_version;

ALTER TABLE ai_usage_events ALTER COLUMN ledger_id DROP NOT NULL;
ALTER TABLE ai_usage_events ALTER COLUMN user_id DROP NOT NULL;

ALTER TABLE wardrobe_items ADD COLUMN task_id UUID;
CREATE INDEX idx_wardrobe_items_task_id ON wardrobe_items(task_id);

ALTER TABLE brand_items ADD COLUMN task_id UUID;
CREATE INDEX idx_brand_items_task_id ON brand_items(task_id);

INSERT INTO ai_model_configs (provider, model, input_usd_per_million, output_usd_per_million, usd_to_vnd)
VALUES ('google', 'gemini-3.1-flash-lite', 0.25, 1.50, 27500),
       ('google', 'gemini-2.5-flash-lite', 0.10, 0.40, 27500),
       ('google', 'gemini-embedding-2', 0.20, 0.20, 27500),
       ('google', 'gemini-embedding-001', 0.15, 0.15, 27500),
       ('google', 'gemma-4-31b-it', 0, 0, 27500),
       ('google', 'gemma-4-26b-a4b-it', 0, 0, 27500);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_brand_items_task_id;
ALTER TABLE brand_items DROP COLUMN IF EXISTS task_id;

DROP INDEX IF EXISTS idx_wardrobe_items_task_id;
ALTER TABLE wardrobe_items DROP COLUMN IF EXISTS task_id;

ALTER TABLE ai_usage_events ALTER COLUMN ledger_id SET NOT NULL;
ALTER TABLE ai_usage_events ALTER COLUMN user_id SET NOT NULL;

DELETE FROM ai_model_configs WHERE provider = 'google' AND model IN (
  'gemini-3.1-flash-lite', 'gemini-2.5-flash-lite',
  'gemini-embedding-2', 'gemini-embedding-001',
  'gemma-4-31b-it', 'gemma-4-26b-a4b-it'
);

ALTER TABLE ai_usage_events
    ADD COLUMN pricing_version VARCHAR(100),
    DROP COLUMN pricing_config_id;

ALTER TABLE ai_policy_grants RENAME TO user_ai_policy_grants;

DROP TABLE IF EXISTS ai_model_configs;
-- +goose StatementEnd
```
