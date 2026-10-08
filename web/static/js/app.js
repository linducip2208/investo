/* Investo app shell JS (local, no CDN). Vanilla. Works with server-rendered HTML. */
(function () {
  "use strict";

  var App = (window.Investo = window.Investo || {});

  /* ---------------- theme: light | dark | system (default light) ---------------- */
  var THEME_KEY = "investo-theme";
  function systemDark() {
    return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
  }
  function resolveTheme(pref) {
    if (pref === "dark") return "dark";
    if (pref === "system") return systemDark() ? "dark" : "light";
    return "light";
  }
  App.getThemePref = function () {
    try { return localStorage.getItem(THEME_KEY) || "light"; } catch (e) { return "light"; }
  };
  App.setThemePref = function (pref) {
    try { localStorage.setItem(THEME_KEY, pref); } catch (e) {}
    applyTheme(pref);
    updateThemeButtons(pref);
  };
  function applyTheme(pref) {
    var mode = resolveTheme(pref);
    document.documentElement.classList.toggle("dark", mode === "dark");
    document.documentElement.setAttribute("data-theme", mode);
    // Tabler v1 dark mode hook (data-bs-theme) + native color-scheme
    try {
      document.documentElement.setAttribute("data-bs-theme", mode);
      document.documentElement.style.colorScheme = mode;
    } catch (e) {}
    try {
      var meta = document.querySelector('meta[name="theme-color"]');
      if (meta) meta.setAttribute("content", mode === "dark" ? "#141b2e" : "#f4f6fb");
    } catch (e) {}
    document.dispatchEvent(new CustomEvent("investo:theme", { detail: { mode: mode, pref: pref } }));
  }
  App.currentMode = function () { return resolveTheme(App.getThemePref()); };
  function updateThemeButtons(pref) {
    document.querySelectorAll("[data-theme-menu] [data-set-theme]").forEach(function (btn) {
      var active = btn.getAttribute("data-set-theme") === pref;
      btn.setAttribute("aria-pressed", active ? "true" : "false");
      btn.classList.toggle("is-active", active);
    });
  }
  // apply ASAP (also mirrored by inline head script to avoid FOUC)
  applyTheme(App.getThemePref());
  if (window.matchMedia) {
    try {
      window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", function () {
        if (App.getThemePref() === "system") applyTheme("system");
      });
    } catch (e) {}
  }
  document.addEventListener("click", function (ev) {
    var btn = ev.target.closest("[data-set-theme]");
    if (btn) App.setThemePref(btn.getAttribute("data-set-theme"));
  });

  /* ---------------- sidebar drawer (mobile) ---------------- */
  App.toggleSidebar = function (open) {
    var aside = document.getElementById("app-sidebar");
    var overlay = document.getElementById("app-sidebar-overlay");
    if (!aside) return;
    var willOpen = typeof open === "boolean" ? open : aside.classList.contains("is-closed");
    aside.classList.toggle("is-closed", !willOpen);
    aside.setAttribute("aria-hidden", willOpen ? "false" : "true");
    if (overlay) overlay.classList.toggle("open", willOpen);
    var toggles = document.querySelectorAll("[data-sidebar-toggle]");
    toggles.forEach(function (t) { t.setAttribute("aria-expanded", willOpen ? "true" : "false"); });
  };
  document.addEventListener("click", function (ev) {
    if (ev.target.closest("[data-sidebar-toggle]")) { App.toggleSidebar(); return; }
    if (ev.target.closest("#app-sidebar-overlay")) { App.toggleSidebar(false); return; }
    // close dropdowns on outside click
    document.querySelectorAll("[data-dropdown].open").forEach(function (dd) {
      if (!ev.target.closest("[data-dropdown]")) dd.classList.remove("open");
    });
  });
  document.addEventListener("keydown", function (ev) {
    if (ev.key === "Escape") {
      App.toggleSidebar(false);
      document.querySelectorAll("[data-dropdown].open").forEach(function (dd) { dd.classList.remove("open"); });
      var mb = document.querySelector(".fin-modal-backdrop");
      if (mb && mb.dataset.esc !== "off") mb.remove();
    }
  });
  document.addEventListener("click", function (ev) {
    var t = ev.target.closest("[data-dropdown-toggle]");
    if (t) {
      var dd = t.closest("[data-dropdown]");
      if (dd) dd.classList.toggle("open");
    }
  });

  /* ---------------- toasts ---------------- */
  var ICONS = {
    success: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>',
    error: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>',
    danger: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>',
    warning: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01M10.3 3.9L1.8 18a2 2 0 001.7 3h17a2 2 0 001.7-3L13.7 3.9a2 2 0 00-3.4 0z"/></svg>',
    info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>'
  };
  App.toast = function (type, message, title) {
    var wrap = document.getElementById("fin-toasts");
    if (!wrap) {
      wrap = document.createElement("div");
      wrap.id = "fin-toasts";
      wrap.setAttribute("role", "status");
      wrap.setAttribute("aria-live", "polite");
      document.body.appendChild(wrap);
    }
    var t = (type === "error") ? "danger" : (type || "info");
    var el = document.createElement("div");
    el.className = "fin-toast " + t;
    el.innerHTML = (ICONS[t] || ICONS.info) + "<div><strong></strong><div></div></div>";
    el.querySelector("strong").textContent = title || ({ success: "Berhasil", danger: "Terjadi kesalahan", warning: "Perhatian", info: "Info" }[t] || "Info");
    el.querySelector("div div").textContent = message;
    wrap.appendChild(el);
    setTimeout(function () { el.style.opacity = "0"; el.style.transition = "opacity .3s"; setTimeout(function () { el.remove(); }, 320); }, 4200);
  };

  /* ---------------- fetch helper with human errors ---------------- */
  var STATUS_MSG = {
    401: "Sesi berakhir. Silakan masuk kembali.",
    403: "Anda tidak memiliki akses untuk tindakan ini.",
    404: "Data tidak ditemukan.",
    422: "Data tidak valid. Periksa kembali isian formulir.",
    429: "Terlalu banyak permintaan. Coba lagi sebentar."
  };
  App.api = async function (url, opts) {
    var res;
    try {
      res = await fetch(url, opts);
    } catch (e) {
      throw new Error("Jaringan bermasalah. Periksa koneksi internet Anda.");
    }
    if (res.ok) return res;
    if (res.status >= 500) throw new Error("Layanan sedang gangguan. Coba lagi beberapa saat.");
    throw new Error(STATUS_MSG[res.status] || ("Permintaan gagal (kode " + res.status + ")."));
  };
  App.debounce = function (fn, ms) {
    var t; return function () {
      var a = arguments, c = this;
      clearTimeout(t); t = setTimeout(function () { fn.apply(c, a); }, ms || 250);
    };
  };
  App.fmtIDR = function (n) {
    if (n === null || n === undefined || isNaN(n)) return "—";
    try { return "Rp " + new Intl.NumberFormat("id-ID").format(Math.round(n)); } catch (e) { return "Rp " + n; }
  };
  App.fmtPct = function (n) {
    if (n === null || n === undefined || isNaN(n)) return "—";
    var s = (n > 0 ? "+" : "") + Number(n).toFixed(2) + "%";
    return s;
  };

  /* ---------------- charts (local lightweight-charts, theme aware) ---------------- */
  function cssVar(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name);
    return (v && v.trim()) || fallback;
  }
  App.chartTheme = function () {
    var dark = document.documentElement.classList.contains("dark");
    return {
      layout: { background: { type: "solid", color: "transparent" }, textColor: cssVar("--ui-muted", dark ? "#a3b1cc" : "#5b6b82"), fontFamily: "Inter, system-ui, sans-serif", fontSize: 11 },
      grid: { vertLines: { color: cssVar("--ui-border", dark ? "#2c3658" : "#e3e8f0") }, horzLines: { color: cssVar("--ui-border", dark ? "#2c3658" : "#e3e8f0") } },
      up: cssVar("--ui-success", "#18794e"), down: cssVar("--ui-danger", "#c93a3a"),
      line: cssVar("--ui-primary", "#2065d1")
    };
  };
  App.lineChart = function (el, series, opts) {
    if (!window.LightweightCharts || !el) return null;
    opts = opts || {};
    var t = App.chartTheme();
    var chart = LightweightCharts.createChart(el, {
      autoSize: true, height: opts.height || 260,
      layout: t.layout, grid: t.grid,
      rightPriceScale: { borderColor: t.grid.horzLines.color },
      timeScale: { borderColor: t.grid.horzLines.color }
    });
    var line = chart.addLineSeries({ color: opts.color || t.line, lineWidth: 2, priceLineVisible: false });
    line.setData(series || []);
    var refill = function () {
      var nt = App.chartTheme();
      chart.applyOptions({ layout: nt.layout, grid: nt.grid });
    };
    document.addEventListener("investo:theme", refill);
    return { chart: chart, series: line };
  };
  // tiny canvas sparkline, no dependency
  App.sparkline = function (canvas, values, up) {
    if (!canvas || !values || !values.length) return;
    var dpr = window.devicePixelRatio || 1;
    var w = canvas.clientWidth || 96, h = canvas.clientHeight || 28;
    canvas.width = w * dpr; canvas.height = h * dpr;
    var ctx = canvas.getContext("2d"); ctx.scale(dpr, dpr);
    var min = Math.min.apply(null, values), max = Math.max.apply(null, values);
    var span = (max - min) || 1;
    var dark = document.documentElement.classList.contains("dark");
    ctx.strokeStyle = up === undefined ? cssVar("--ui-primary", "#2065d1") : (up ? (dark ? "#4cc38a" : "#18794e") : (dark ? "#f26d6d" : "#c93a3a"));
    ctx.lineWidth = 1.6; ctx.lineJoin = "round"; ctx.beginPath();
    values.forEach(function (v, i) {
      var x = (i / (values.length - 1)) * (w - 4) + 2;
      var y = h - 3 - ((v - min) / span) * (h - 6);
      if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
    });
    ctx.stroke();
  };
  App.paintSparklines = function (root) {
    (root || document).querySelectorAll("canvas.spark[data-points]").forEach(function (c) {
      try {
        var pts = JSON.parse(c.getAttribute("data-points"));
        var upAttr = c.getAttribute("data-up");
        App.sparkline(c, pts, upAttr === null ? undefined : upAttr === "1");
      } catch (e) {}
    });
  };

  /* ---------------- i18n (lightweight shell dictionary) ---------------- */
  var LANG_KEY = "investo-lang";
  var STR = {
    en: { nav_dashboard: "Dashboard", nav_portfolio: "Portfolio", nav_markets: "Markets", nav_watchlist: "Watchlist", nav_transactions: "Transactions", nav_analysis: "Analysis", nav_reports: "Reports", nav_settings: "Settings", search_ph: "Search symbol, market…", theme: "Theme", theme_light: "Light", theme_dark: "Dark", theme_system: "System", empty_title: "No data yet", error_title: "Something went wrong", retry: "Try again" },
    id: { nav_dashboard: "Dasbor", nav_portfolio: "Portofolio", nav_markets: "Pasar", nav_watchlist: "Watchlist", nav_transactions: "Transaksi", nav_analysis: "Analisis", nav_reports: "Laporan", nav_settings: "Pengaturan", search_ph: "Cari simbol, pasar…", theme: "Tema", theme_light: "Terang", theme_dark: "Gelap", theme_system: "Sistem", empty_title: "Belum ada data", error_title: "Terjadi gangguan", retry: "Coba lagi" }
  };
  App.getLang = function () { try { return localStorage.getItem(LANG_KEY) || "id"; } catch (e) { return "id"; } };
  App.setLang = function (l) { try { localStorage.setItem(LANG_KEY, l); } catch (e) {} App.applyLang(l); };
  App.applyLang = function (l) {
    var lang = l || App.getLang();
    var dict = STR[lang] || STR.id;
    document.querySelectorAll("[data-i18n]").forEach(function (el) {
      var k = el.getAttribute("data-i18n");
      if (dict[k]) el.textContent = dict[k];
    });
    document.querySelectorAll("[data-i18n-ph]").forEach(function (el) {
      var k = el.getAttribute("data-i18n-ph");
      if (dict[k]) el.setAttribute("placeholder", dict[k]);
    });
    document.documentElement.setAttribute("lang", lang === "en" ? "en" : "id");
  };
  document.addEventListener("click", function (ev) {
    var b = ev.target.closest("[data-set-lang]");
    if (b) App.setLang(b.getAttribute("data-set-lang"));
  });

  /* ---------------- PWA ---------------- */
  App.registerSW = function () {
    if ("serviceWorker" in navigator) {
      window.addEventListener("load", function () {
        navigator.serviceWorker.register("/static/sw.js").catch(function () {});
      });
    }
  };
  function offlineBanner() {
    var update = function () {
      var off = !navigator.onLine;
      document.body.classList.toggle("is-offline", off);
      var bar = document.getElementById("offline-bar");
      if (off && !bar) {
        bar = document.createElement("div");
        bar.id = "offline-bar";
        bar.setAttribute("role", "alert");
        bar.style.cssText = "position:sticky;top:0;z-index:120;background:#b7791f;color:#fff;text-align:center;font-size:13px;padding:8px 12px;";
        bar.textContent = "Anda sedang offline. Menampilkan data terakhir yang tersimpan.";
        document.body.prepend(bar);
      } else if (!off && bar) { bar.remove(); }
    };
    window.addEventListener("online", update);
    window.addEventListener("offline", update);
    update();
  }

  /* ---------------- boot ---------------- */
  document.addEventListener("DOMContentLoaded", function () {
    updateThemeButtons(App.getThemePref());
    App.applyLang();
    App.paintSparklines();
    App.registerSW();
    offlineBanner();
  });
})();
