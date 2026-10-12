const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/saham", { waitUntil: "domcontentloaded", timeout: 45000 });
  await page.waitForTimeout(1000);
  const m = await page.evaluate(() => {
    const aside = document.querySelector("aside.sidebar");
    const out = { start: getComputedStyle(aside).display, sheets: [] };
    const sheets = [...document.querySelectorAll('link[rel="stylesheet"], style')];
    sheets.forEach((el, i) => {
      const label = el.tagName + ":" + (el.getAttribute("href") || el.getAttribute("data-investo-compat") !== null ? "compat" : "page-style").slice(0, 40);
      out.sheets.push(label);
    });
    // disable each style source one at a time
    const els = [...document.querySelectorAll('link[rel="stylesheet"], style')];
    const res = [];
    els.forEach((el) => {
      el.disabled = true;
      res.push(getComputedStyle(aside).display);
      el.disabled = false;
    });
    out.afterEachDisable = res;
    return out;
  });
  console.log(JSON.stringify(m));
  await browser.close();
})();
