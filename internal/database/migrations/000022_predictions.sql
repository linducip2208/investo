CREATE TABLE IF NOT EXISTS price_predictions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    stock_code VARCHAR(10) NOT NULL,
    predicted_price DECIMAL(15,2) NOT NULL,
    predicted_date DATE NOT NULL,
    actual_price DECIMAL(15,2),
    accuracy DECIMAL(10,4),
    points INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_predictions_user (user_id),
    INDEX idx_predictions_date (predicted_date),
    INDEX idx_predictions_code (stock_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
