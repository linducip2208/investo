const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1366, height: 768 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(2000);
  const out = await page.evaluate(() => {
    const res = {};
    const docW = () => document.documentElement.scrollWidth;
    res.base = docW();
    const main = document.querySelector("main");
    const kids = [...main.children];
    // hide each top-level child one at a time
    kids.forEach((k, i) => {
      k.style.display = "none";
      res["without_" + i + "_" + k.tagName + "_" + String(k.className).slice(0, 30)] = docW();
      k.style.display = "";
    });
    return res;
  });
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})();
