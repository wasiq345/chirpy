-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP not null,
    updated_at TIMESTAMP not null,
    Email TEXT not null UNIQUE
);

-- +goose Down
DROP TABLE users;