-- +goose Up
-- +goose StatementBegin
ALTER TABLE user ADD COLUMN totp_secret VARCHAR(255) DEFAULT '';
ALTER TABLE user ADD COLUMN totp_enabled BOOL DEFAULT FALSE NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user DROP COLUMN totp_secret;
ALTER TABLE user DROP COLUMN totp_enabled;
-- +goose StatementEnd
