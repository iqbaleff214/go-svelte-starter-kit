CREATE TYPE email_status AS ENUM ('queued', 'sent', 'failed', 'dead');

CREATE TABLE email_logs (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    template   VARCHAR(100) NOT NULL,
    recipient  VARCHAR(255) NOT NULL,
    status     email_status NOT NULL DEFAULT 'queued',
    error      TEXT,
    attempts   SMALLINT NOT NULL DEFAULT 0,
    sent_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_logs_user_id ON email_logs (user_id);
CREATE INDEX idx_email_logs_status ON email_logs (status);
CREATE INDEX idx_email_logs_created_at ON email_logs (created_at DESC);
