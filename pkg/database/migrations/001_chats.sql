-- +migrate Up
CREATE TABLE chats (
    id BIGINT NOT NULL,
    topic_id INTEGER NOT NULL,
    text_model VARCHAR(255) NOT NULL,
    system_prompt TEXT,
    image_model VARCHAR(255) NOT NULL,
    ttl BIGINT NOT NULL,
    messages JSONB,
    last_update TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, topic_id)
);

CREATE INDEX idx_active_chats
    ON chats (id, topic_id, last_update);