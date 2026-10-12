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
  const info = await page.evaluate(() => {
    const bar = document.querySelector(".animate-ticker");
    if (!bar) return { none: true };
    const parent = bar.parentElement;
    const pr = parent.getBoundingClientRect();
    const br = bar.getBoundingClientRect();
    const kids = [...bar.children].slice(0, 3).map((el) => {
      const r = el.getBoundingClientRect();
      return { tag: el.tagName, cls: el.className, top: Math.round(r.top), bottom: Math.round(r.bottom), h: Math.round(r.height), text: el.textContent.slice(0, 40) };
    });
    return {
      parentTop: Math.round(pr.top), parentH: Math.round(pr.height),
      barTop: Math.round(br.top), barH: Math.round(br.height),
      n: bar.children.length, kids,
    };
  });
  console.log(JSON.stringify(info, null, 1));
  await browser.close();
})();
