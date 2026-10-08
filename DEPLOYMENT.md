# Investo — Deployment Guide

Panduan deploy aplikasi Investo (Go + MySQL) ke server produksi Linux.

## Prasyarat
- Linux server (Ubuntu/Debian/Alpine) + MySQL 8.0+ (atau 8.4) + Redis
- `mysql-client` untuk backup terjadwal melalui `mysqldump`
- Go 1.26 (untuk build) — atau cukup upload binary jadi

## 1. Build binary (self-contained)

Binary sudah meng-embed template, static, dan migrasi — **tidak butuh folder `web/` atau `internal/`**.

```bash
# Windows → Linux (cross-compile)
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -o investo-linux ./cmd/server/

# Atau langsung di Linux
CGO_ENABLED=0 go build -o investo-linux ./cmd/server/
```

## 2. Setup direktori

```bash
sudo useradd --system --home /opt/investo --shell /usr/sbin/nologin investo
sudo install -d -o investo -g investo -m 0750 /opt/investo /opt/investo/data/backup
# upload investo-linux + .env ke /opt/investo/
cd /opt/investo
chmod +x investo-linux
sudo chown investo:investo investo-linux .env
sudo chmod 0600 .env
```

## 3. Konfigurasi `.env`

```env
INVESTO_PORT=8000
INVESTO_DB_HOST=127.0.0.1
INVESTO_DB_PORT=3306
INVESTO_DB_USER=investo
INVESTO_DB_PASS=password-acak-yang-kuat
INVESTO_DB_NAME=investo
INVESTO_REDIS_ADDR=127.0.0.1:6379
INVESTO_APP_NAME=Investo
INVESTO_APP_URL=https://investo.whitelabel.co.id
INVESTO_APP_ENV=production
INVESTO_JWT_SECRET=rahasia-acak-minimal-32-karakter
INVESTO_SESSION_SECRET=rahasia-acak-berbeda-minimal-32-karakter

# AI (opsional) — atau via /pengaturan/ai
AI_BASE_URL=https://api.openai.com/v1
AI_API_KEY=
AI_MODEL=gpt-4o-mini

# Data fallback (opsional)
FMP_API_KEY=
```

Pada mode `production`, proses menolak start jika password database kosong,
kedua secret kurang dari 32 karakter/masih placeholder/sama, atau `APP_URL` bukan
URL HTTPS absolut. Buat secret secara independen, misalnya dengan
`openssl rand -hex 32`, dan jangan commit file `.env`.

### Alternatif Docker Compose

Salin `.env.example` menjadi `.env`, isi seluruh nilai wajib dengan secret unik,
lalu jalankan `docker compose up -d --build`. MySQL dan Redis hanya tersedia di
jaringan internal Compose; port aplikasi secara default hanya bind ke
`127.0.0.1:8000` untuk diproxy oleh Nginx.

## 4. Jalankan (systemd)

```bash
cp deploy/investo.service /etc/systemd/system/investo.service
systemctl daemon-reload
systemctl enable --now investo
systemctl status investo
```

## 5. Nginx reverse proxy

```bash
cp deploy/nginx.conf /etc/nginx/sites-available/investo
ln -s /etc/nginx/sites-available/investo /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

## 6. Verifikasi

- `systemctl status investo` → `active (running)`
- Log: `journalctl -u investo -f` → harus muncul `Migrations completed` + `Templates parsed successfully`
- `/health` → 200
- `/indexnow-key.txt` → key 32 karakter

## Migrasi database

Migrasi berjalan **otomatis** saat server start (di-embed dalam binary). Startup
berhenti jika migrasi gagal. Advisory lock MySQL mencegah dua instance menjalankan
migrasi bersamaan; periksa log sebelum melakukan rollback binary.

## Backup database

Backup otomatis tiap hari (24 jam) via `mysqldump` → `data/backup/` (simpan 14 file terakhir).
Salin backup secara terenkripsi ke storage di luar server dan lakukan uji restore
berkala; file lokal saja tidak cukup untuk disaster recovery.

## Troubleshooting

| Masalah | Solusi |
|---|---|
| `address already in use` | `ss -tlnp \| grep 8000` lalu kill proses lama |
| `Migrations completed` tidak muncul | cek kredensial DB di `.env` |
| Template error | binary lama — rebuild/deploy ulang |
| Data tidak update | cek scheduler di log; cron-fetch binary |
