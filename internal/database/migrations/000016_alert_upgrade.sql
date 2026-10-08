SET @col_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'alerts' AND COLUMN_NAME = 'conditions_json');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE alerts ADD COLUMN conditions_json LONGTEXT AFTER target_price', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'alerts' AND COLUMN_NAME = 'notification_type');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE alerts ADD COLUMN notification_type VARCHAR(50) DEFAULT ''web'' AFTER conditions_json', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'alerts' AND COLUMN_NAME = 'trigger_count');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE alerts ADD COLUMN trigger_count INT DEFAULT 0 AFTER notification_type', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'alerts' AND COLUMN_NAME = 'last_triggered_at');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE alerts ADD COLUMN last_triggered_at TIMESTAMP NULL AFTER trigger_count', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
