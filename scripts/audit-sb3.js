const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/saham", { waitUntil: "domcontentloaded", timeout: 45000 });
  await page.waitForTimeout(800);
  const m = await page.evaluate(() => {
    const out = [];
    for (const sh of document.styleSheets) {
      let href;
      try { href = sh.href || "inline"; } catch (e) { continue; }
      const walk = (list) => {
        for (const r of list) {
          if (r.cssRules) { walk(r.cssRules); continue; }
          if (r.selectorText && /(^|[^\w-])\.d-flex([^\w-]|$)|(^|[^\w-])\.sidebar([^\w-]|$)/.test(r.selectorText)) {
            out.push(href.split("/").pop() + " :: " + r.selectorText.slice(0, 90) + " => display:" + r.style.display + " pri:" + r.style.getPropertyPriority("display"));
          }
        }
      };
      try { walk(sh.cssRules); } catch (e) {}
    }
    return out;
  });
  console.log(m.join("\n"));
  await browser.close();
})();
