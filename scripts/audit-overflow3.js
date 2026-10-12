const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1366, height: 768 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(2000);
  const out = await page.evaluate(() => {
    const content = document.querySelector("body > div");
    const rows = [];
    for (const el of [content, ...content.children]) {
      const cs = getComputedStyle(el);
      rows.push({
        tag: el.tagName, cls: String(el.className).slice(0, 50),
        w: Math.round(el.getBoundingClientRect().width),
        sw: el.scrollWidth,
        kidsw: [...el.children].map((c) => Math.round(c.getBoundingClientRect().width)).join(","),
      });
    }
    // check legacy-compat + ml-60 rule presence
    let ml60 = "absent";
    for (const sh of document.styleSheets) {
      try {
        for (const r of sh.cssRules) {
          if (r.selectorText && r.selectorText.includes(".ml-60")) ml60 = r.selectorText.slice(0, 80);
        }
      } catch (e) {}
    }
    return { rows, ml60 };
  });
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})();
