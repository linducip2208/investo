CREATE TABLE IF NOT EXISTS ai_response_ratings (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    usage_log_id BIGINT,
    user_id BIGINT,
    rating INT DEFAULT 0,
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (usage_log_id) REFERENCES ai_usage_log(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
