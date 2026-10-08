SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'alerts' AND INDEX_NAME = 'idx_alerts_user_active');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE alerts ADD INDEX idx_alerts_user_active (user_id, is_active)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'notifications' AND INDEX_NAME = 'idx_notifications_user_read_created');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE notifications ADD INDEX idx_notifications_user_read_created (user_id, is_read, created_at)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'stock_prices' AND INDEX_NAME = 'idx_stock_prices_stock_date');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE stock_prices ADD INDEX idx_stock_prices_stock_date (stock_id, date)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'portfolio_items' AND INDEX_NAME = 'idx_portfolio_items_portfolio');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE portfolio_items ADD INDEX idx_portfolio_items_portfolio (portfolio_id)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'paper_trades' AND INDEX_NAME = 'idx_paper_trades_portfolio');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE paper_trades ADD INDEX idx_paper_trades_portfolio (portfolio_id)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ai_usage_log' AND INDEX_NAME = 'idx_ai_usage_user_created');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE ai_usage_log ADD INDEX idx_ai_usage_user_created (user_id, created_at)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'news' AND INDEX_NAME = 'idx_news_published');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE news ADD INDEX idx_news_published (published_at)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'blog_posts' AND INDEX_NAME = 'idx_blog_posts_cat_published');
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE blog_posts ADD INDEX idx_blog_posts_cat_published (category_id, is_published)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt
