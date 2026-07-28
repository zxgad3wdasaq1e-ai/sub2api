-- Durable metadata for the built-in image studio. Binary image data remains in
-- the configured S3-compatible object storage; PostgreSQL stores only keys and
-- generation metadata.
CREATE TABLE IF NOT EXISTS image_generation_jobs (
    task_id VARCHAR(96) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    prompt TEXT NOT NULL DEFAULT '',
    model VARCHAR(255) NOT NULL DEFAULT '',
    mode VARCHAR(32) NOT NULL DEFAULT 'text',
    size VARCHAR(64) NOT NULL DEFAULT '',
    quality VARCHAR(32) NOT NULL DEFAULT '',
    output_format VARCHAR(16) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'processing',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT image_generation_jobs_status_check CHECK (status IN ('processing', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_image_generation_jobs_user_created
    ON image_generation_jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_image_generation_jobs_expires
    ON image_generation_jobs (expires_at);

CREATE TABLE IF NOT EXISTS image_assets (
    asset_id VARCHAR(128) PRIMARY KEY,
    task_id VARCHAR(96) NOT NULL REFERENCES image_generation_jobs(task_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    image_index INTEGER NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    content_type VARCHAR(128) NOT NULL,
    byte_size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT image_assets_task_index_unique UNIQUE (task_id, image_index)
);

CREATE INDEX IF NOT EXISTS idx_image_assets_user_created
    ON image_assets (user_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_image_assets_expires
    ON image_assets (expires_at)
    WHERE deleted_at IS NULL;
