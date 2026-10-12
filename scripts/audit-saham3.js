const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await ctx.newPage();
  const errs = [];
  page.on("pageerror", (e) => errs.push(String(e && e.message || e).slice(0, 150)));
  page.on("console", (m) => { if (m.type() === "error") errs.push("CONSOLE:" + m.text().slice(0, 150)); });
  await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(4000);
  const m = await page.evaluate(async () => {
    const mq = matchMedia("(max-width:1023px)").matches;
    let sidebarRule = "not-found";
    for (const sh of document.styleSheets) {
      try {
        for (const r of sh.cssRules) {
          if (r.conditionText && r.conditionText.includes("1023")) {
            for (const ir of r.cssRules) {
              if (ir.selectorText && ir.selectorText.includes(".sidebar")) sidebarRule = ir.selectorText + " -> " + ir.style.display;
            }
          }
        }
      } catch (e) {}
    }
    let apiOk = "?";
    try {
      const r = await fetch("/api/market/summary");
      const d = await r.json();
      apiOk = "status=" + r.status + " ticker=" + ((d.ticker || []).length);
    } catch (e) { apiOk = "FETCH-FAIL:" + e.message; }
    return { mq, sidebarRule, apiOk };
  });
  console.log(JSON.stringify(m));
  console.log("errs:", errs.length ? errs.join(" || ") : "(none)");
  await browser.close();
})();
