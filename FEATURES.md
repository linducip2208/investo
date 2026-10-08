# Investo — AI-Powered Stock & Forex Platform

Platform analisis saham IDX & forex Indonesia dengan **AI multi-agent system**, 500+ saham, real-time dashboard, dan 200+ fitur.

## Tech Stack
- **Backend**: Go 1.26 + Chi Router + MySQL 8.4 + sqlx
- **Frontend**: TailwindCSS + Alpine.js + Lightweight Charts v4
- **AI**: BYOK (DeepSeek / OpenAI / Claude) + Multi-Agent System
- **Real-time**: WebSocket + 5-minute scheduler
- **Deploy**: Windows + Linux binary, Docker, docker-compose

## Quick Start
```bash
# Setup
copy .env.example .env
mysql -u root -e "CREATE DATABASE investo CHARACTER SET utf8mb4"

# Run
go build -o investo.exe ./cmd/server/
./investo.exe

# Seed data
.\fetch.exe          # 500+ stocks + prices
.\backfill.exe       # Historical data
.\fundamental.exe    # Fundamental data
```

Open `http://localhost:8000` — Login: `admin@investo.test` / `password`

---

## Feature List

### 📊 Dashboard & Market
| Feature | Route |
|---|---|
| Bloomberg-Style Dashboard | `/dashboard` |
| Live Ticker Bar | Dashboard |
| Market Breadth (Advance/Decline) | Dashboard |
| Candlestick Chart IHSG | Dashboard |
| Real-time WebSocket | Dashboard |
| Notification Center | Dashboard |
| Global Search (Cmd+K) | Dashboard |
| Dark Mode Toggle | All pages |
| Mobile Responsive | All pages |

### 📈 Stocks (124 IDX)
| Feature | Route |
|---|---|
| Stock List with Search/Filter | `/saham` |
| Stock Detail (Price, Chart, Fundamentals) | `/saham/{code}` |
| Technical Analysis (Candlestick + 6 Indicators) | `/saham/{code}/teknikal` |
| Chart Pro (Drawing Tools, Multi-TF) | `/saham/{code}/chart-pro` |
| Fundamental Analysis | `/saham/{code}/fundamental` |
| Valuation (DCF, Graham, Lynch, PBV) | `/saham/{code}/valuasi` |
| Multi-Timeframe Grid | `/saham/{code}/multi-tf` |
| Liquidity Zones | Stock detail |
| Stock Similarity | Stock detail |
| Insider Transaction | `/market/insider` |

### 💱 Forex (12 Pairs)
| Feature | Route |
|---|---|
| Forex List | `/forex` |
| Forex Detail | `/forex/{base}-{quote}` |
| Forex Pro Terminal | `/forex/pro` |
| Currency Strength Meter | Pro Terminal |
| Carry Trade Calculator | Pro Terminal |
| Correlation Matrix | Pro Terminal |
| Arbitrage Spotter | Pro Terminal |
| Fibonacci Calculator | Pro Terminal |

### 🔍 Screener & Analysis
| Feature | Route |
|---|---|
| Stock Screener (15+ filters) | `/screener` |
| Natural Language Screener | `/screener` (AI) |
| Quick Presets (6 strategies) | `/screener` |
| Market Heatmap | `/market/heatmap` |
| Fear & Greed Index | `/market/fear-greed` |
| Sector Rotation Radar | `/market/sector-rotation` |
| Sector Health Index | `/market/sector-health` |
| Pair Trading Finder | `/market/pair-trading` |
| Correlation Matrix | `/market/correlation` |
| Pattern Scanner | `/market/pattern-scan` |
| Trend Scanner | `/market/trend-scanner` |
| Price Action Monitor | `/market/price-action` |
| Liquidity Zones | Stock detail |

### 📰 News & Data
| Feature | Route |
|---|---|
| News Feed | `/berita` |
| Market Recap | `/market/recap` |
| Economic Calendar | `/market/economic-calendar` |
| Dividend Calendar | `/dividen` |
| IPO Watch | `/market/ipo` |
| Index Rebalance Predictor | `/market/rebalance` |
| Foreign Flow Tracker | `/market/foreign-flow` |
| Syariah Compliance | `/market/syariah` |
| Waran Tracker | `/market/waran` |
| Bandarmologi Detector | `/market/bandarmologi` |
| Saham Gorengan Alert | `/market/saham-gorengan` |
| Market Anomalies | `/market/anomalies` |

### 💼 Portfolio
| Feature | Route |
|---|---|
| Portfolio CRUD | `/dashboard/portfolios` |
| Portfolio Analytics | `/dashboard/portfolios/{id}/analytics` |
| Portfolio Tools (Monte Carlo, Stress Test) | `/dashboard/portfolios/tools` |
| Portfolio Risk Analysis | `/dashboard/portfolios/{id}/risk` |
| Tax-Loss Harvesting | Portfolio tools |
| DRIP Calculator | Portfolio tools |
| CSV Import | Portfolio page |
| Profit/Loss Calculator | Portfolio detail |
| Portfolio Weather | Dashboard + Portfolio |
| Shared Portfolio | `/shared/{token}` |

### ⭐ Watchlist & Alerts
| Feature | Route |
|---|---|
| Watchlist CRUD | `/dashboard/watchlists` |
| Multi-Condition Alerts | `/dashboard/alerts` |
| Alert Templates (8 presets) | Alerts page |
| Alert History | Alerts page |
| Telegram Notifications | Settings |
| Webhook Alerts | `/api/webhooks` |

### 👥 Community
| Feature | Route |
|---|---|
| Investment Ideas | `/ideas` |
| Idea Comments & Likes | `/ideas/{id}` |
| Stock Discussions | `/saham/{code}/diskusi` |
| Leaderboard | `/leaderboard` |
| Prediction Market | `/market/prediction` |
| Achievement System | `/achievements` |

### 🤖 AI Tools (BYOK)
| Feature | Route |
|---|---|
| AI Provider Settings (DeepSeek/OpenAI/Claude) | `/pengaturan/ai` |
| Daily Market Briefing | `/ai/briefing` |
| AI Stock Research Report | `/ai/report/{code}` |
| AI Stock Comparison | `/ai/compare` |
| AI Trading Coach Chat | `/ai/coach` |
| AI Investment Thesis Builder | `/ai/thesis` |
| AI Smart Search | `/ai/search` |
| AI Risk Test | `/ai/risk-test` |
| AI Report Generator | `/ai/report-generator` |
| AI Social Post Generator | `/ai/social` |

### 🧠 Multi-Agent System
| Feature | Route |
|---|---|
| 9-Agent Analysis | `/ai/agents` |
| Agent Personas (4 styles) | Agent dashboard |
| Consensus Voting | Agent dashboard |
| Streaming Live Output | API (SSE) |
| Decision Memory | Auto |
| Signal Confidence Scoring | API |
| Signal Backtesting | API |
| Market Regime Detection | API |
| Optimal Entry Finder | API |
| Price Range Forecast | API |
| Black Swan Warning | API |

### 🎨 Creative AI
| Feature | Route |
|---|---|
| Voice-to-Trade | `/ai/voice-trade` |
| Daily Trader Standup | `/ai/standup` |
| Market Comic Generator | `/ai/comic` |
| Portfolio Haiku | API |
| Stock Story Generator | `/ai/stock-story/{code}` |

### 🏢 Enterprise Tools
| Feature | Route |
|---|---|
| Tax Optimizer | `/ai/tax-optimizer` |
| OJK Compliance Checker | `/ai/compliance` |
| Client Report Generator | `/ai/client-report` |
| Webhook Intelligence | `/ai/webhooks` |
| Calendar Sync (.ics) | API |
| Slack/Discord Bot Format | API |

### 🔧 Calculators
| Feature | Route |
|---|---|
| Calculator Hub | `/calculators` |
| Right Issue Calculator | `/calculators/right-issue` |
| Tax Calculator | `/calculators/tax` |
| Fibonacci Calculator | Chart tools |

### 🛠️ Admin
| Feature | Route |
|---|---|
| User Management | `/admin/users` |
| Stock Management | `/admin/stocks` |
| News Management | `/admin/news` |
| Blog Management | `/admin/blog` |
| Sector Management | `/admin/sectors` |
| System Settings | `/admin/settings` |
| Data Pipeline Dashboard | `/admin/pipeline` |
| Database Backup | `/admin/backup` |
| API Key Management | `/admin/settings/api-key` |

### 📡 API
| Feature | Route |
|---|---|
| REST API v1 (14 endpoints) | `/api/v1/*` |
| JWT Authentication | `/api/v1/auth/login` |
| Interactive API Docs | `/api/docs` |
| Webhook Receiver | `/api/webhooks/alert` |
| Health Check | `/health` |
| WebSocket | `/ws` |

### 🎨 Design
| Feature | Detail |
|---|---|
| Dark Theme | All pages (#0b1120 background) |
| Gradient Cards | Dashboard stat cards |
| SignalAIX-Inspired | Professional trading floor aesthetic |
| Responsive Layout | Mobile + Tablet + Desktop |
| Loading Skeletons | All data-driven pages |
| Custom Error Pages | 404 + 500 |

### 🗺️ SEO
| Feature | Route |
|---|---|
| Glossary (80+ terms) | `/istilah/{slug}` |
| City Pages (100+ cities) | `/belajar-saham-{city}` |
| How-To Guides | `/cara/{slug}` |
| Sitemap.xml | `/sitemap.xml` |
| Robots.txt | `/robots.txt` |

### 📝 Content
| Feature | Route |
|---|---|
| Blog System | `/blog` |
| Landing Page | `/` |
| Documentation | `/docs` |
| Pricing Page | `/pricing` |

### 🔒 Security
| Feature | Detail |
|---|---|
| Session Auth | Cookie-based |
| JWT API Auth | `/api/v1/*` |
| Rate Limiting | 60 req/min/IP |
| API Key Auth | Query param |

### 🐳 DevOps
| Feature | Detail |
|---|---|
| Docker | Dockerfile + docker-compose.yml |
| Linux Binary | `investo-linux` |
| Graceful Shutdown | SIGINT/SIGTERM |
| Structured Logging | JSON format |
| Unit Tests | 18 tests (indicator, pattern, sentiment) |

---

## Data Pipeline
- **Source**: Yahoo Finance API + RSS News
- **Frequency**: Every 5 minutes (trading hours 09:00-16:00 WIB)
- **Coverage**: 124 IDX stocks + 12 forex pairs
- **History**: 1 year daily data
- **Fundamental**: Estimated from price data (with AI augmentation)

## Project Structure
```
investo/
├── cmd/
│   ├── server/main.go      # Main server entry
│   ├── fetch/main.go       # Data fetcher
│   ├── backfill/main.go    # Historical backfill
│   └── fundamental/main.go # Fundamental generator
├── internal/
│   ├── config/             # Configuration
│   ├── database/           # DB connection + migrations
│   ├── handler/            # HTTP handlers (20+ files)
│   ├── middleware/          # Auth, CORS, rate limit, JWT
│   ├── model/              # Data models
│   ├── repository/         # Database access layer
│   ├── service/            # Business logic (30+ services)
│   │   ├── ai_*.go         # AI services
│   │   ├── indicator/      # Technical indicators
│   │   ├── pattern/        # Pattern recognition
│   │   ├── scraper/        # Yahoo + IDX scraper
│   │   └── pseo/           # SEO generator
│   └── pseo/               # PSEO routes
├── web/
│   ├── templates/          # HTML templates (100+ files)
│   │   ├── ai/             # AI feature pages
│   │   ├── admin/          # Admin panel
│   │   ├── auth/           # Login/Register
│   │   ├── blog/           # Blog
│   │   ├── calculators/    # Calculator tools
│   │   ├── community/      # Ideas, Leaderboard
│   │   ├── components/     # Shared components
│   │   ├── dashboard/      # Bloomberg dashboard
│   │   ├── dividen/        # Dividend calendar
│   │   ├── docs/           # Documentation
│   │   ├── forex/          # Forex pages
│   │   ├── market/         # Market analysis pages
│   │   ├── news/           # News pages
│   │   ├── pages/          # Static pages
│   │   ├── portfolio/      # Portfolio pages
│   │   ├── pseo/           # SEO pages
│   │   ├── screener/       # Screener
│   │   ├── settings/       # Settings
│   │   └── stocks/         # Stock pages
│   └── static/             # CSS, JS
├── data/backup/            # Database backups
├── scripts/                # Utility scripts
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

## AI Configuration
1. Buka `/pengaturan/ai`
2. Masukkan API Key dari provider:
   - **DeepSeek**: `https://api.deepseek.com/v1`
   - **OpenAI**: `https://api.openai.com/v1`
   - **Claude**: `https://api.anthropic.com/v1`
3. Klik "Set Active" → "Test Connection"
4. Semua fitur AI siap digunakan

## Default Accounts
| Role | Email | Password |
|---|---|---|
| Admin | admin@investo.test | password |
| User | demo@investo.test | password |

## Build Commands
```bash
# Windows
go build -o investo.exe ./cmd/server/

# Linux
GOOS=linux GOARCH=amd64 go build -o investo-linux ./cmd/server/

# Docker
docker build -t investo .
docker compose up
```

## License
Source code marketplace — whitelabel ready.
