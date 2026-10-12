const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1366, height: 768 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(2000);
  const out = await page.evaluate(() => {
    const q = (s) => document.querySelector(s);
    const info = (el) => {
      if (!el) return null;
      const cs = getComputedStyle(el);
      const r = el.getBoundingClientRect();
      return { w: Math.round(r.width), minW: cs.minWidth, flex: cs.flex, marginL: cs.marginLeft, overflowX: cs.overflowX, overflowY: cs.overflowY, scrollW: el.scrollWidth };
    };
    const main = q("main");
    const kids = [...main.children].map((el) => ({
      tag: el.tagName, cls: String(el.className).slice(0, 60),
      w: Math.round(el.getBoundingClientRect().width),
      sw: el.scrollWidth, minW: getComputedStyle(el).minWidth,
    }));
    return {
      bodyW: Math.round(document.body.getBoundingClientRect().width),
      contentDiv: info(q("body > div")),
      main: info(main),
      kids,
    };
  });
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})();
