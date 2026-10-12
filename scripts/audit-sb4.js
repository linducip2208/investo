const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await ctx.newPage();
  page.on("console", (m) => console.log("BROWSER:", m.text().slice(0, 120)));
  page.on("requestfailed", (r) => console.log("REQFAIL:", r.url(), r.failure() && r.failure().errorText));
  await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(1500);
  const m = await page.evaluate(() => {
    const arr = [];
    for (let i = 0; i < document.styleSheets.length; i++) {
      const sh = document.styleSheets[i];
      let n = -1;
      try { n = sh.cssRules.length; } catch (e) { n = -2; }
      arr.push((sh.href || "inline") + " rules=" + n);
    }
    return arr;
  });
  console.log("SHEETS:", JSON.stringify(m));
  await browser.close();
})();
