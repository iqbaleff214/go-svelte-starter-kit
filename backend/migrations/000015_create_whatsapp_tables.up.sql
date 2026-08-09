-- WhatsApp sessions: one row per linked WA device
CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT        NOT NULL,
    phone        TEXT        NOT NULL DEFAULT '',
    jid          TEXT        NOT NULL DEFAULT '',   -- populated after pairing
    status       TEXT        NOT NULL DEFAULT 'pending', -- pending|connected|disconnected|banned
    paused       BOOLEAN     NOT NULL DEFAULT FALSE,
    last_used_at TIMESTAMPTZ,
    sent_today   INT         NOT NULL DEFAULT 0,
    created_by   UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wa_sessions_status ON whatsapp_sessions(status);
CREATE INDEX idx_wa_sessions_jid    ON whatsapp_sessions(jid);

-- WhatsApp message log / send queue
CREATE TABLE IF NOT EXISTS whatsapp_messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID        REFERENCES whatsapp_sessions(id) ON DELETE SET NULL,
    recipient  TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    status     TEXT        NOT NULL DEFAULT 'queued', -- queued|sent|failed
    error      TEXT        NOT NULL DEFAULT '',
    queued_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at    TIMESTAMPTZ,
    created_by UUID        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_wa_messages_status  ON whatsapp_messages(status);
CREATE INDEX idx_wa_messages_session ON whatsapp_messages(session_id);
CREATE INDEX idx_wa_messages_queued  ON whatsapp_messages(queued_at DESC);
