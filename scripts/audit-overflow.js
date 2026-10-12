const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1366, height: 768 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(2000);
  const out = await page.evaluate(() => {
    const res = [];
    function desc(el, depth) {
      if (depth > 4 || !el.getBoundingClientRect) return;
      const r = el.getBoundingClientRect();
      if (r.width > 1150) {
        res.push(`${"  ".repeat(depth)}<${el.tagName.toLowerCase()} class="${(el.className.baseVal !== undefined ? el.className.baseVal : el.className || "").toString().slice(0, 90)}" w=${Math.round(r.width)}`);
        [...el.children].forEach((c) => desc(c, depth + 1));
      }
    }
    desc(document.querySelector("main"), 0);
    return res;
  });
  console.log(out.join("\n"));
  await browser.close();
})();
