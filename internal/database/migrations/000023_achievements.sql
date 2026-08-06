CREATE TABLE IF NOT EXISTS user_achievements (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    achievement_key VARCHAR(50) NOT NULL,
    progress INT DEFAULT 0,
    unlocked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uq_user_achievement (user_id, achievement_key),
    INDEX idx_ua_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
