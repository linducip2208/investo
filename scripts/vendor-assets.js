#!/usr/bin/env node
/* Copies vendored frontend assets from node_modules into web/static.
 * Run: npm run assets
 * Tabler CSS itself is compiled via: npm run css:tabler (needs `sass`).
 */
const fs = require("fs");
const path = require("path");

const ROOT = path.resolve(__dirname, "..");
function copy(src, dst) {
  const from = path.join(ROOT, src);
  const to = path.join(ROOT, dst);
  fs.mkdirSync(path.dirname(to), { recursive: true });
  fs.copyFileSync(from, to);
  console.log("copied", src, "->", dst);
}

copy("node_modules/alpinejs/dist/cdn.min.js", "web/static/js/vendor/alpine.min.js");
copy(
  "node_modules/lightweight-charts/dist/lightweight-charts.standalone.production.js",
  "web/static/js/vendor/lightweight-charts.js"
);
copy("node_modules/chart.js/dist/chart.umd.min.js", "web/static/js/vendor/chart.umd.min.js");
copy("node_modules/marked/lib/marked.umd.js", "web/static/js/vendor/marked.min.js");

// Tabler outline icons used by the app shell (see web/static/icons).
const ICONS = [
  "layout-dashboard", "briefcase", "chart-line", "chart-bar", "chart-pie", "eye", "bell",
  "settings", "user", "search", "sun", "moon", "menu-2", "trending-up", "trending-down",
  "wallet", "building-bank", "coins", "cash", "exchange", "file-text", "logout", "x",
  "chevron-down", "plus", "trash", "download", "printer", "alert-circle", "alert-triangle",
  "info-circle", "circle-check", "inbox", "arrow-up-right", "arrow-down-right", "calendar",
  "filter", "star", "bulb", "trophy", "calculator", "map", "ruler", "bolt", "link",
  "target", "robot", "brain", "refresh", "sparkles", "shield", "clock", "message-circle",
  "atom", "leaf", "scale", "notebook", "microscope", "package", "credit-card", "school",
  "news", "rss", "book", "medal", "microphone", "diamond", "clipboard-list", "pencil",
  "ticket", "world", "satellite", "droplet", "swords", "history", "mood-smile", "heart",
  "rocket", "books", "hammer", "ghost",
];
let missing = 0;
for (const name of ICONS) {
  const from = path.join(ROOT, "node_modules/@tabler/icons/icons/outline", name + ".svg");
  if (!fs.existsSync(from)) {
    console.warn("missing icon:", name);
    missing++;
    continue;
  }
  copy(
    "node_modules/@tabler/icons/icons/outline/" + name + ".svg",
    "web/static/icons/" + name + ".svg"
  );
}
if (missing > 0) process.exitCode = 1;
