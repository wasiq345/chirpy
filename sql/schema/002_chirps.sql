-- +goose Up
CREATE TABLE chirp(
    id UUID PRIMARY KEY,
    created_at TIMESTAMP not null,
    updated_at TIMESTAMP not null,
    body TEXT not null,
    userId UUID not null,
    CONSTRAINT fk_id FOREIGN KEY (userId) REFERENCES users (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirp;