CREATE TABLE IF NOT EXISTS subscription_plans (
    code VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price_monthly BIGINT DEFAULT 0,
    features TEXT,
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO subscription_plans (code, name, price_monthly) VALUES
('free', 'Free', 0),
('pro', 'Pro', 49000),
('whitelabel', 'Whitelabel', 499000);

CREATE TABLE IF NOT EXISTS payments (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    order_id VARCHAR(64) UNIQUE NOT NULL,
    plan VARCHAR(50),
    amount BIGINT DEFAULT 0,
    status VARCHAR(30) DEFAULT 'pending',
    raw_callback LONGTEXT,
    paid_at DATETIME NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_payments_user_status (user_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
