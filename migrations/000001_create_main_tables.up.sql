-- migrations/000001_create_main_tables.up.sql
CREATE TABLE hacking_infos (
    id BIGSERIAL PRIMARY KEY,
    protocol VARCHAR(255) NOT NULL,
    network VARCHAR(255) NOT NULL,
    amount VARCHAR(255) NOT NULL,
    tx_hash VARCHAR(255) NOT NULL,
    report_time TIMESTAMPTZ NOT NULL,
    message_id INT NOT NULL
);

CREATE TABLE transfer_infos (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(255) NOT NULL,
    amount VARCHAR(255) NOT NULL,
    from_address VARCHAR(255) NOT NULL,
    to_address VARCHAR(255) NOT NULL,
    report_time TIMESTAMPTZ NOT NULL,
    message_id INT NOT NULL
);

CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE telegram_channel (
    username VARCHAR(255) PRIMARY KEY,
    last_message_id INT NOT NULL
);