/* Capture real product screenshots for tutorial/docs (local only). */
const { chromium } = require("playwright-core");
const fs = require("fs");
const path = require("path");

const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
const OUT = path.join(__dirname, "..", "web", "static", "screenshots");
fs.mkdirSync(OUT, { recursive: true });

const PAGES = [
  { name: "login", url: "/login" },
  { name: "saham", url: "/saham" },
  { name: "forex", url: "/forex" },
  { name: "screener", url: "/screener" },
  { name: "docs", url: "/docs" },
];

(async () => {
  const browser = await chromium.launch();
  const results = [];
  for (const vp of [
    { w: 1440, h: 900, suffix: "desktop" },
    { w: 390, h: 844, suffix: "mobile" },
  ]) {
    const ctx = await browser.newContext({ viewport: { width: vp.w, height: vp.h } });
    const page = await ctx.newPage();
    for (const p of PAGES) {
      try {
        await page.goto(BASE + p.url, { waitUntil: "networkidle", timeout: 30000 });
        await page.waitForTimeout(1200);
        const file = path.join(OUT, `${p.name}-${vp.suffix}.png`);
        await page.screenshot({ path: file });
        results.push(`OK ${p.name}-${vp.suffix}`);
      } catch (e) {
        results.push(`FAIL ${p.name}-${vp.suffix}: ${e.message.split("\n")[0]}`);
      }
    }
    // authenticated dashboard via demo account
    try {
      await page.goto(BASE + "/login", { waitUntil: "networkidle", timeout: 30000 });
      await page.fill('input[name="email"]', "demo@investo.test");
      await page.fill('input[name="password"]', "password");
      await Promise.all([
        page.waitForNavigation({ timeout: 15000 }).catch(() => {}),
        page.click('button[type="submit"]'),
      ]);
      await page.waitForTimeout(2000);
      const file = path.join(OUT, `dashboard-${vp.suffix}.png`);
      await page.screenshot({ path: file });
      results.push(`OK dashboard-${vp.suffix}`);
    } catch (e) {
      results.push(`FAIL dashboard-${vp.suffix}: ${e.message.split("\n")[0]}`);
    }
    await ctx.close();
  }
  await browser.close();
  console.log(results.join("\n"));
})();
