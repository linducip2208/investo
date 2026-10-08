CREATE TABLE IF NOT EXISTS agent_run_checkpoints (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    run_date DATE NOT NULL,
    phase VARCHAR(30) NOT NULL,
    reports_json LONGTEXT,
    debate_log TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_ticker_date_phase (ticker, run_date, phase)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
