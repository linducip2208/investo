# Investo

Platform analisis saham IDX & forex Indonesia dengan **multi-agent AI**, 500+ saham, dan 200+ fitur.

## Tech Stack
- **Backend**: Go 1.26 + Chi Router + MySQL 8.4 + sqlx + Redis
- **Frontend**: TailwindCSS + Alpine.js + Lightweight Charts v4
- **AI**: BYOK multi-provider (12 provider) + Multi-Agent System (9 agent)
- **Deploy**: single self-contained binary (Windows + Linux) + Docker

## Quick Start

```bash
# 1. Setup
copy .env.example .env   # isi DB + APP_URL

# 2. Database
mysql -u root -e "CREATE DATABASE investo CHARACTER SET utf8mb4"

# 3. Run (migrasi otomatis saat start)
go run ./cmd/server/
# atau build binary
go build -o investo.exe ./cmd/server/

# 4. Seed data (opsional)
go run ./cmd/fetch/        # saham + harga
go run ./cmd/fundamental/  # data fundamental
```

Buka `http://localhost:8000` — Login: `admin@investo.test` / `password`

## Fitur Utama
- **Multi-Agent AI** (`/ai/agents`) — 9 agent → BUY/SELL/HOLD + entry/target/stop
- **Screener** 15+ filter + natural language
- **Portfolio & Watchlist** + alert harga
- **Forex + Crypto** (8 pasangan)
- **Valuasi deterministik**: DCF (bull/bear), DDM, Graham, Lynch, Monte Carlo
- **Programmatic SEO** — 1 juta+ halaman (`/best-*`, `/alternatives-to-*`, `/compare/*`, dll.)

## SEO — Sitemap & IndexNow

### Sitemap
- `/sitemap.xml` (dinamis, auto-generate dari DB)
- Submit ke Google Search Console: **Search Console → Sitemaps → masukkan `sitemap.xml`**

### IndexNow (auto-submit)
Sudah built-in — URL baru otomatis di-submit ke Bing, Yandex, Seznam, Naver tiap hari.

- Key: `/indexnow-key.txt`
- Submit manual: `POST https://api.indexnow.org/indexnow` dengan payload:
```json
{ "host": "investo.whitelabel.co.id", "key": "<key>", "keyLocation": "https://investo.whitelabel.co.id/indexnow-key.txt", "urlList": ["https://investo.whitelabel.co.id/blog/artikel-baru"] }
```

## AI Configuration (BYOK)
1. Buka `/pengaturan/ai`
2. Masukkan API key provider (OpenAI, DeepSeek, Claude, Gemini, Groq, dll.)
3. Klik "Test Connection" → semua fitur AI siap

## Deployment
Lihat [DEPLOYMENT.md](DEPLOYMENT.md) — panduan lengkap (binary self-contained, systemd, nginx, supervisor, backup).

## Lisensi
Source code marketplace — whitelabel ready.
