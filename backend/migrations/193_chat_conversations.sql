-- Server-authoritative chat history. Every user-owned row carries user_id so
-- repositories can enforce tenant isolation in the predicate itself.
CREATE TABLE IF NOT EXISTS chat_conversations (
    conversation_id VARCHAR(128) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT 'New conversation',
    model VARCHAR(255) NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_updated
    ON chat_conversations (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS chat_messages (
    message_id VARCHAR(128) PRIMARY KEY,
    conversation_id VARCHAR(128) NOT NULL REFERENCES chat_conversations(conversation_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    parent_id VARCHAR(128),
    role VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'completed',
    parts_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    token_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_messages_role_check CHECK (role IN ('system', 'user', 'assistant')),
    CONSTRAINT chat_messages_status_check CHECK (status IN ('pending', 'completed', 'failed', 'cancelled'))
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_created
    ON chat_messages (conversation_id, created_at, message_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_user_conversation
    ON chat_messages (user_id, conversation_id);

CREATE TABLE IF NOT EXISTS chat_runs (
    run_id VARCHAR(128) PRIMARY KEY,
    conversation_id VARCHAR(128) NOT NULL REFERENCES chat_conversations(conversation_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    user_message_id VARCHAR(128) NOT NULL REFERENCES chat_messages(message_id) ON DELETE CASCADE,
    assistant_message_id VARCHAR(128) NOT NULL REFERENCES chat_messages(message_id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    idempotency_key VARCHAR(255) NOT NULL DEFAULT '',
    model VARCHAR(255) NOT NULL DEFAULT '',
    api_key_id BIGINT,
    usage_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_runs_status_check CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_runs_user_idempotency
    ON chat_runs (user_id, idempotency_key) WHERE idempotency_key <> '';
CREATE INDEX IF NOT EXISTS idx_chat_runs_user_created
    ON chat_runs (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS chat_attachments (
    attachment_id VARCHAR(128) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    content_type VARCHAR(128) NOT NULL,
    byte_size BIGINT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chat_attachments_user_created
    ON chat_attachments (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_attachments_expires
    ON chat_attachments (expires_at);
