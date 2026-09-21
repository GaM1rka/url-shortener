-- +goose Up
CREATE TABLE links (
    short_code VARCHAR(10) NOT NULL,
    original_url TEXT NOT NULL,
    CONSTRAINT links_pkey PRIMARY KEY (short_code),
    CONSTRAINT links_short_code_format CHECK (short_code ~ '^[A-Za-z0-9_]{10}$'),
    CONSTRAINT links_original_url_key UNIQUE (original_url)
);

-- +goose Down
DROP TABLE links;
