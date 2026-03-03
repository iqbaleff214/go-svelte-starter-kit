CREATE TABLE ai_conversations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(255),
    messages    JSONB NOT NULL DEFAULT '[]',
    model       VARCHAR(100) NOT NULL,
    token_usage INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_conversations_user_id ON ai_conversations (user_id);
CREATE INDEX idx_ai_conversations_updated_at ON ai_conversations (updated_at DESC);
