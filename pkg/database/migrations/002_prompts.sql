-- +migrate Up
CREATE TABLE prompts (
    id SERIAL PRIMARY KEY,
    text TEXT NOT NULL
);