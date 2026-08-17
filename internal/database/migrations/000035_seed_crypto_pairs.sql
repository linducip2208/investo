INSERT INTO forex_pairs (base_currency, quote_currency, name, `group`) VALUES
('BTC','USD','Bitcoin / US Dollar','crypto'),
('ETH','USD','Ethereum / US Dollar','crypto'),
('BNB','USD','BNB / US Dollar','crypto'),
('XRP','USD','XRP / US Dollar','crypto'),
('SOL','USD','Solana / US Dollar','crypto'),
('ADA','USD','Cardano / US Dollar','crypto'),
('DOGE','USD','Dogecoin / US Dollar','crypto'),
('AVAX','USD','Avalanche / US Dollar','crypto')
ON DUPLICATE KEY UPDATE name=VALUES(name), `group`=VALUES(`group`);
