CREATE TABLE IF NOT EXISTS trade_journal (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    stock_code VARCHAR(10) NOT NULL,
    entry_date DATE NOT NULL,
    exit_date DATE,
    entry_price DECIMAL(15,2) NOT NULL,
    exit_price DECIMAL(15,2),
    quantity DECIMAL(15,4) NOT NULL,
    direction VARCHAR(10) NOT NULL DEFAULT 'buy',
    strategy_used VARCHAR(255),
    notes TEXT,
    emotions VARCHAR(50),
    outcome VARCHAR(20),
    profit_loss DECIMAL(15,2),
    profit_loss_pct DECIMAL(10,4),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
