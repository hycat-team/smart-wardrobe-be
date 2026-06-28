-- +goose Up
CREATE TABLE IF NOT EXISTS brand_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
    last_message_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT uq_brand_conversations_brand_user UNIQUE (brand_id, user_id),
    CONSTRAINT chk_brand_conversations_status CHECK (status IN ('OPEN', 'CLOSED'))
);

CREATE TABLE IF NOT EXISTS brand_conversation_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES brand_conversations(id) ON DELETE CASCADE,
    sender_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    sender_role VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_brand_conversation_messages_sender_role CHECK (sender_role IN ('CUSTOMER', 'BRAND_STAFF', 'SYSTEM'))
);

CREATE INDEX IF NOT EXISTS idx_brand_conversations_brand_status ON brand_conversations(brand_id, status);
CREATE INDEX IF NOT EXISTS idx_brand_conversation_messages_conv_created ON brand_conversation_messages(conversation_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_brand_conversation_messages_conv_created;
DROP INDEX IF EXISTS idx_brand_conversations_brand_status;
DROP TABLE IF EXISTS brand_conversation_messages;
DROP TABLE IF EXISTS brand_conversations;
