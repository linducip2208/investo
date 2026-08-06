INSERT INTO sectors (name, slug, description) VALUES
('Energi', 'energi', 'Sektor energi, minyak, gas, dan batubara'),
('Bahan Baku', 'bahan-baku', 'Sektor pertambangan, kimia, dan material dasar'),
('Industri', 'industri', 'Sektor manufaktur dan industri'),
('Konsumen Primer', 'konsumen-primer', 'Sektor barang konsumsi primer'),
('Konsumen Non-Primer', 'konsumen-non-primer', 'Sektor barang konsumsi non-primer'),
('Kesehatan', 'kesehatan', 'Sektor farmasi dan layanan kesehatan'),
('Keuangan', 'keuangan', 'Sektor perbankan, asuransi, dan jasa keuangan'),
('Properti & Real Estat', 'properti-real-estat', 'Sektor properti dan real estat'),
('Teknologi', 'teknologi', 'Sektor teknologi informasi'),
('Infrastruktur', 'infrastruktur', 'Sektor infrastruktur, transportasi, dan utilitas'),
('Transportasi & Logistik', 'transportasi-logistik', 'Sektor transportasi dan logistik')
ON DUPLICATE KEY UPDATE name=VALUES(name);
