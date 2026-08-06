CREATE TABLE IF NOT EXISTS stocks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    sector_id BIGINT,
    subsector VARCHAR(255),
    listing_date DATE,
    shares_outstanding BIGINT DEFAULT 0,
    description TEXT,
    logo_url VARCHAR(500),
    website VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (sector_id) REFERENCES sectors(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS stock_prices (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    stock_id BIGINT NOT NULL,
    date DATE NOT NULL,
    open DECIMAL(15,2) DEFAULT 0,
    high DECIMAL(15,2) DEFAULT 0,
    low DECIMAL(15,2) DEFAULT 0,
    close DECIMAL(15,2) DEFAULT 0,
    volume BIGINT DEFAULT 0,
    adj_close DECIMAL(15,2) DEFAULT 0,
    UNIQUE KEY uk_stock_date (stock_id, date),
    FOREIGN KEY (stock_id) REFERENCES stocks(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS stock_fundamentals (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    stock_id BIGINT NOT NULL,
    period VARCHAR(20) NOT NULL,
    report_type VARCHAR(20) NOT NULL DEFAULT 'annual',
    revenue DECIMAL(30,2) DEFAULT 0,
    net_income DECIMAL(30,2) DEFAULT 0,
    eps DECIMAL(15,4) DEFAULT 0,
    bvps DECIMAL(15,4) DEFAULT 0,
    total_assets DECIMAL(30,2) DEFAULT 0,
    total_liabilities DECIMAL(30,2) DEFAULT 0,
    equity DECIMAL(30,2) DEFAULT 0,
    roe DECIMAL(10,4) DEFAULT 0,
    roa DECIMAL(10,4) DEFAULT 0,
    per DECIMAL(10,4) DEFAULT 0,
    pbv DECIMAL(10,4) DEFAULT 0,
    der DECIMAL(10,4) DEFAULT 0,
    net_profit_margin DECIMAL(10,4) DEFAULT 0,
    dividend_yield DECIMAL(10,4) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_stock_period_type (stock_id, period, report_type),
    FOREIGN KEY (stock_id) REFERENCES stocks(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS stock_actions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    stock_id BIGINT NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    ex_date DATE,
    record_date DATE,
    payment_date DATE,
    description TEXT,
    ratio DECIMAL(10,4) DEFAULT 0,
    price DECIMAL(15,2) DEFAULT 0,
    FOREIGN KEY (stock_id) REFERENCES stocks(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
