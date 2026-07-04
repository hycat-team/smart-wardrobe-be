-- +goose Up

CREATE TABLE ai_pricing_configs (
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
    ADD COLUMN pricing_config_id UUID REFERENCES ai_pricing_configs(id),
    DROP COLUMN pricing_version;

-- +goose Down

ALTER TABLE ai_usage_events
    ADD COLUMN pricing_version VARCHAR(100),
    DROP COLUMN pricing_config_id;

ALTER TABLE ai_policy_grants RENAME TO user_ai_policy_grants;

DROP TABLE IF EXISTS ai_pricing_configs;
