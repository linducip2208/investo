CREATE TABLE IF NOT EXISTS feature_flags (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT TRUE,
    description TEXT,
    plan_required VARCHAR(50) DEFAULT 'free',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO feature_flags (name, enabled, plan_required, description) VALUES
('paper_trading', TRUE, 'free', 'Paper trading simulator'),
('ai_signals', TRUE, 'pro', 'AI trading signals'),
('multi_agent', TRUE, 'pro', 'Multi-agent analysis'),
('backtesting', TRUE, 'pro', 'Strategy backtesting'),
('screener_advanced', TRUE, 'pro', 'Advanced screener filters'),
('api_access', TRUE, 'whitelabel', 'API v1 access'),
('export_pdf', TRUE, 'pro', 'PDF export'),
('whatsapp_alerts', TRUE, 'pro', 'WhatsApp notifications'),
('custom_indicators', TRUE, 'pro', 'Custom indicator builder'),
('dark_mode', TRUE, 'free', 'Dark mode toggle');
