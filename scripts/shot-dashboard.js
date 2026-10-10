const { chromium } = require("playwright-core");
const path = require("path");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
const OUT = path.join(__dirname, "..", "web", "static", "screenshots");
(async () => {
  const browser = await chromium.launch();
  for (const vp of [{ w: 1440, h: 900, suffix: "desktop" }]) {
    const ctx = await browser.newContext({ viewport: { width: vp.w, height: vp.h } });
    const page = await ctx.newPage();
    await page.goto(BASE + "/login", { waitUntil: "networkidle", timeout: 30000 });
    await page.fill('input[name="email"]', "demo@investo.test");
    await page.fill('input[name="password"]', "password");
    await Promise.all([
      page.waitForNavigation({ timeout: 15000 }).catch(() => {}),
      page.click('button[type="submit"]'),
    ]);
    // wait for IHSG chart canvas to actually render
    await page.waitForFunction(
      () => {
        const el = document.getElementById("main-chart");
        return el && el.querySelector("canvas");
      },
      { timeout: 25000 }
    ).catch(() => {});
    await page.waitForTimeout(2500);
    await page.screenshot({ path: path.join(OUT, `dashboard-${vp.suffix}.png`) });
    console.log("dashboard retake done");
    await ctx.close();
  }
  await browser.close();
})();
