-- +goose Up
-- Создание таблицы users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,                
    user_name VARCHAR(50) UNIQUE NOT NULL,   
    user_password TEXT NOT NULL,   
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание индекса на поле user_name
CREATE INDEX idx_users_user_name ON users(user_name);

-- Создание таблицы user_tokens
CREATE TABLE user_tokens (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL UNIQUE, 
    token TEXT UNIQUE NOT NULL,
    access BOOL DEFAULT FALSE, -- (если true - токену дан доступ к БД)
    created_at TIMESTAMP NOT NULL,
    expired_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_tokens_user_id ON user_tokens(user_id);

-- Создание таблицы orders
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,                
    user_id INT NOT NULL,                 
    order_number TEXT NOT NULL, 
    order_status TEXT NOT NULL,               -- (NEW, PROCESSING, INVALID, PROCESSED)
    order_accrual DECIMAL(10, 2) DEFAULT 0.0, -- баллы Accrual
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (user_id, order_number),
    UNIQUE (order_number)
);

CREATE INDEX idx_orders_user_id ON orders(user_id);

-- Создание таблицы queue_order (сохранение заказов, если Accrual недоступен)
CREATE TABLE queue_order (
    id SERIAL PRIMARY KEY,                                
    order_number TEXT NOT NULL, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (order_number)
);

-- Создание таблицы balance
CREATE TABLE balance (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE NOT NULL,  
    accrual DECIMAL(10, 2) DEFAULT 0.0,        -- баллы Accrual
    withdrawn DECIMAL(10, 2) DEFAULT 0.0,      -- сумма использованных за весь период регистрации баллов
    FOREIGN KEY (user_id) REFERENCES users(id)  
);

CREATE INDEX idx_balance_user_id ON balance(user_id);

-- Создание таблицы withdrawals
CREATE TABLE withdrawals (
    id SERIAL PRIMARY KEY,  
    user_id INT NOT NULL,  
    order_number TEXT NOT NULL, 
    sum DECIMAL(10, 2) DEFAULT 0.0,           -- баллы Accrual  
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,  
    UNIQUE (user_id, order_number)
);

CREATE INDEX idx_withdrawals_user_id ON withdrawals(user_id);

-- +goose Down
-- Удаление таблицы orders
DROP TABLE IF EXISTS orders;

-- Удаление таблицы balance
DROP TABLE IF EXISTS balance;

-- Удаление таблицы queue_order
DROP TABLE IF EXISTS queue_order;

-- Удаление таблицы withdrawals
DROP TABLE IF EXISTS withdrawals;

-- Удаление таблицы user_tokens
DROP TABLE IF EXISTS user_tokens;

-- Удаление таблицы users
DROP TABLE IF EXISTS users;

-- Удаление индексов
DROP INDEX IF EXISTS idx_users_user_name; 
DROP INDEX IF EXISTS idx_user_tokens_user_id;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP INDEX IF EXISTS idx_balance_user_id;
DROP INDEX IF EXISTS idx_withdrawals_user_id;
