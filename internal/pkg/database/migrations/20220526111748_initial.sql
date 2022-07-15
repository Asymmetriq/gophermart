-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id UUID UNIQUE NOT NULL PRIMARY KEY,
    login TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
    user_id uuid UNIQUE NOT NULL PRIMARY KEY,
    current_balance float NOT NULL DEFAULT 0,
    withdrawn float NOT NULL DEFAULT 0
);

DROP TYPE IF EXISTS status;
CREATE TYPE status AS ENUM ('NEW', 'PROCESSING', 'PROCESSED', 'INVALID');

CREATE TABLE IF NOT EXISTS orders (
    order_number text UNIQUE NOT NULL PRIMARY KEY, 
    user_id uuid NOT NULL ,
    order_status status NOT NULL DEFAULT 'NEW',
    accrual float, 
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS withdrawals (
    order_number text UNIQUE NOT NULL PRIMARY KEY,
    user_id uuid NOT NULL,
    sum float NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP DATABASE users;
DROP DATABASE balances;
DROP DATABASE orders;
DROP DATABASE withdrawals;
-- +goose StatementEnd
