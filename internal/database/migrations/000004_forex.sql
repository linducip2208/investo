CREATE TABLE IF NOT EXISTS forex_pairs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    base_currency VARCHAR(10) NOT NULL,
    quote_currency VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    `group` VARCHAR(50) NOT NULL DEFAULT 'major',
    UNIQUE KEY uk_pair (base_currency, quote_currency)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS forex_rates (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    pair_id BIGINT NOT NULL,
    date DATE NOT NULL,
    open DECIMAL(15,6) DEFAULT 0,
    high DECIMAL(15,6) DEFAULT 0,
    low DECIMAL(15,6) DEFAULT 0,
    close DECIMAL(15,6) DEFAULT 0,
    UNIQUE KEY uk_pair_date (pair_id, date),
    FOREIGN KEY (pair_id) REFERENCES forex_pairs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
