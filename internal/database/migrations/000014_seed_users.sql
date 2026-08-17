INSERT INTO users (name, email, password, role) VALUES
('Admin Investo', 'admin@investo.test', '$2a$10$NVS.UNMEsmjI9VLUbsLXKeFuJqdluXj1GT5FozSyBuwyHdhY6kxOa', 'admin'),
('Demo User', 'demo@investo.test', '$2a$10$NVS.UNMEsmjI9VLUbsLXKeFuJqdluXj1GT5FozSyBuwyHdhY6kxOa', 'user')
ON DUPLICATE KEY UPDATE email=VALUES(email);
