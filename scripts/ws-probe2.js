const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  page.on("response", (r) => {
    if (r.url().endsWith("/ws")) console.log("WS-HTTP:", r.status(), r.headers()["location"] || "");
  });
  await page.goto(BASE + "/login", { waitUntil: "networkidle", timeout: 30000 });
  await page.fill('input[name="email"]', "demo@investo.test");
  await page.fill('input[name="password"]', "password");
  await Promise.all([
    page.waitForNavigation({ timeout: 15000 }).catch(() => {}),
    page.click('button[type="submit"]'),
  ]);
  await page.goto(BASE + "/dashboard", { waitUntil: "networkidle", timeout: 30000 });
  await page.waitForTimeout(4000);
  const ok = await page.evaluate(() => new Promise((resolve) => {
    try {
      const ws = new WebSocket("ws://" + location.host + "/ws");
      ws.onopen = () => { ws.close(); resolve("OPEN"); };
      ws.onerror = () => resolve("ERROR");
      setTimeout(() => resolve("TIMEOUT"), 5000);
    } catch (e) { resolve("THROW:" + e.message); }
  }));
  console.log("WS-RESULT:", ok);
  await browser.close();
})();
