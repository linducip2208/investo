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
  await page.goto(BASE + "/admin", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForTimeout(2500);
  await page.screenshot({ path: path.join(OUT, "admin-dashboard-desktop.png") });
  console.log("admin errs:", errs.splice(0).join(" | ") || "(none)");
  await page.goto(BASE + "/docs", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForTimeout(1500);
  await page.screenshot({ path: path.join(OUT, "docs-desktop.png"), fullPage: false });
  // scroll shot of tutorial section
  await page.locator("#tutorial").scrollIntoViewIfNeeded();
  await page.waitForTimeout(800);
  await page.screenshot({ path: path.join(OUT, "docs-tutorial-desktop.png") });
  console.log("docs errs:", errs.splice(0).join(" | ") || "(none)");
  await browser.close();
})();
