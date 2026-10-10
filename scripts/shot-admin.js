const { chromium } = require("playwright-core");
const path = require("path");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
const OUT = path.join(__dirname, "..", "web", "static", "screenshots");
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  const errs = [];
  page.on("pageerror", (e) => errs.push(String(e && e.message || e).slice(0, 120)));
  await page.goto(BASE + "/login", { waitUntil: "networkidle", timeout: 30000 });
  await page.fill('input[name="email"]', "admin@investo.test");
  await page.fill('input[name="password"]', "password");
  await Promise.all([
    page.waitForNavigation({ timeout: 15000 }).catch(() => {}),
    page.click('button[type="submit"]'),
  ]);
  await page.waitForTimeout(1500);
  for (const [name, url, wait] of [
    ["admin-dashboard", "/admin", 2500],
    ["portfolio", "/dashboard/portfolios", 2500],
    ["watchlist", "/dashboard/watchlists", 2500],
    ["market-heatmap", "/market/heatmap", 3000],
  ]) {
    try {
      await page.goto(BASE + url, { waitUntil: "networkidle", timeout: 30000 });
      await page.waitForTimeout(wait);
      await page.screenshot({ path: path.join(OUT, `${name}-desktop.png`) });
      console.log("OK", name, "errs:", errs.splice(0).join(" | ") || "(none)");
    } catch (e) {
      console.log("FAIL", name, e.message.split("\n")[0]);
    }
  }
  console.log("PAGEERRORS:", errs.length ? errs.join(" || ") : "(none)");
  await browser.close();
})();
