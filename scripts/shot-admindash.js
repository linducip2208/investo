const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/login", { waitUntil: "networkidle", timeout: 30000 });
  await page.fill('input[name="email"]', "admin@investo.test");
  await page.fill('input[name="password"]', "password");
  await Promise.all([
    page.waitForNavigation({ timeout: 15000 }).catch(() => {}),
    page.click('button[type="submit"]'),
  ]);
  await page.waitForTimeout(4000);
  await page.screenshot({ path: "C:\\Users\\LINDUC~1\\AppData\\Local\\Temp\\opencode\\admin-dash.png" });
  console.log("shot done, url=", page.url());
  await browser.close();
})();
