# Frontend — Investo

Fintech UI built on **local Tabler + local Tailwind utilities**. No CDN for UI assets.

## Source of truth

| Asset | Source | Output |
|---|---|---|
| Tabler CSS | `node_modules/@tabler/core/scss` (npm) | `web/static/css/vendor/tabler.css` (compiled with `sass`) |
| Utilities | `web/static/css/input.css` + templates scan | `web/static/css/output.css` (Tailwind v4 CLI) |
| Fintech theme | hand-written | `web/static/css/fintech.css` (tokens, light default, `html.dark` dark mode) |
| Shell JS | hand-written | `web/static/js/app.js` (theme light/dark/system, toasts, charts, i18n shell, PWA) |
| Vendor JS | npm: alpinejs, lightweight-charts, chart.js, marked | `web/static/js/vendor/*` |
| Icons | npm `@tabler/icons` (outline) | `web/static/icons/*.svg` (+ `icon-192/512.png` for PWA) |

Dark mode uses the **class strategy** (`html.dark`, Tailwind `@custom-variant` in `input.css`).
Default theme is **light**; preference persisted as `investo-theme` (light/dark/system).
Shell language persisted as `investo-lang` (id/en) for `[data-i18n]` chrome labels.

## Commands

```bash
npm install            # install frontend deps
npm run build:assets   # vendor JS + icons, compile Tabler CSS + Tailwind
npm run assets         # only copy vendor JS + icons from node_modules
npm run css:tabler     # recompile Tabler SCSS
npm run css:tw         # rebuild Tailwind utilities (run after editing template classes)
go build ./...         # backend (serves web/ via embed, /static/* with cache headers)
go test ./internal/... # backend tests
```

## PWA

- `web/static/manifest.json` (light theme, shortcuts, maskable icon)
- `web/static/sw.js` (`investo-v2`; precaches shell + vendor; API network-only; pages network-first with `/static/offline.html` fallback; push notifications)
- `web/static/offline.html` (offline fallback page)

## Conventions for templates

- No `https://cdn.*` / `unpkg` / Google Fonts in templates — use `/static/*`.
- Finance semantics: `.fin-up` / `.fin-down` + `.fin-tag` (icon + text, never color-only).
- States: `.fin-state` (empty), `.skeleton` (loading), `.fin-alert` (error/info), `Investo.toast()` for notifications.
- Numbers: `.fin-num` / `.mono-num` (tabular), `Investo.fmtIDR` / `Investo.fmtPct`.
- Charts: `Investo.lineChart(el, data)` (local lightweight-charts, theme-aware) or `canvas.spark`.
- API errors: use `Investo.api()` — human-readable id-ID messages for 401/403/404/422/429/5xx/offline.
