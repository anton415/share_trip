-- +goose Up
ALTER TYPE trip_status ADD VALUE IF NOT EXISTS 'started';

-- +goose Down
-- PostgreSQL does not support removing an enum value without recreating the type.
