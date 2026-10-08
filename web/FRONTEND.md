# Frontend — Investo

Fintech UI built on **local Tabler + local Tailwind utilities**. No CDN for UI assets.

## Source of truth

| Asset | Source | Output |
|---|---|---|
| Tabler CSS | `node_modules/@tabler/core/scss` (npm) | `web/static/css/vendor/tabler.css` (compiled with `sass`) |
| Fintech theme | hand-written | `web/static/css/fintech.css` (tokens, light default, `html.dark` + `data-bs-theme` dark mode, documented companions) |
| Legacy states | extracted from local Tailwind build | `web/static/css/legacy-compat.css` (only `dark:`/`disabled:`/`hover:` + leftovers; generated, do not hand-edit) |
| Utilities (build-only) | `web/static/css/input.css` + template scan | `web/static/css/output.css` (source for compat extraction; NOT linked by pages) |
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
npm run build:assets   # vendor JS + icons, compile Tabler CSS + Tailwind + compat
npm run assets         # only copy vendor JS + icons from node_modules
npm run css:tabler     # recompile Tabler SCSS
npm run css:tw         # rebuild Tailwind utilities (build-only source for compat)
npm run css:compat     # regenerate legacy-compat.css from current templates
go build ./...         # backend (serves web/ via embed, /static/* with cache headers)
go test ./internal/... # backend tests
```

Markup uses Tabler component + utility classes (`card`, `btn`, `table`,
`badge bg-*-lt`, `alert`, `empty`, `form-control`, `d-flex`, `row`/`col-*`).
`legacy-compat.css` preserves the remaining state variants Tabler does not
express declaratively (`dark:`, `disabled:`, `hover:`). After editing template
classes, re-run `npm run css:tw && npm run css:compat`.

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
