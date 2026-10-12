const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await ctx.newPage();
  await page.goto(BASE + "/saham", { waitUntil: "domcontentloaded", timeout: 45000 });
  await page.waitForTimeout(1500);
  const m = await page.evaluate(() => {
    const aside = document.querySelector("aside.sidebar");
    // find all rules matching aside, in order
    const matches = [];
    for (const sh of document.styleSheets) {
      let rules;
      try { rules = sh.cssRules; } catch (e) { continue; }
      const walk = (list, media) => {
        for (const r of list) {
          if (r.cssRules) { walk(r.cssRules, r.conditionText || "media"); continue; }
          try {
            if (r.selectorText && aside.matches(r.selectorText.split(",")[0].split(":")[0])) {
              if (r.style.display) matches.push(`${media || "top"} :: ${r.selectorText} => display:${r.style.display}`);
            }
          } catch (e) {}
        }
      };
      walk(rules, null);
    }
    return { cls: aside.className, computed: getComputedStyle(aside).display, matches };
  });
  console.log(JSON.stringify(m, null, 1));
  await browser.close();
})();
