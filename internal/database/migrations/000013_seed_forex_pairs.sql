INSERT INTO forex_pairs (base_currency, quote_currency, name, `group`) VALUES
('EUR','USD','Euro / US Dollar','major'),
('USD','JPY','US Dollar / Japanese Yen','major'),
('GBP','USD','British Pound / US Dollar','major'),
('USD','CHF','US Dollar / Swiss Franc','major'),
('AUD','USD','Australian Dollar / US Dollar','major'),
('USD','CAD','US Dollar / Canadian Dollar','major'),
('NZD','USD','New Zealand Dollar / US Dollar','major'),
('USD','IDR','US Dollar / Indonesian Rupiah','exotic'),
('EUR','IDR','Euro / Indonesian Rupiah','exotic'),
('JPY','IDR','Japanese Yen / Indonesian Rupiah','exotic'),
('SGD','IDR','Singapore Dollar / Indonesian Rupiah','exotic'),
('AUD','IDR','Australian Dollar / Indonesian Rupiah','exotic')
ON DUPLICATE KEY UPDATE name=VALUES(name);
