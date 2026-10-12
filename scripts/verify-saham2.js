const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1366, height: 768 } });
  const page = await ctx.newPage();
  const errs = [];
  page.on("pageerror", (e) => errs.push(String(e && e.message || e).slice(0, 100)));
  await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(5000);
  const before = await page.evaluate(() => {
    const t = document.querySelector(".ticker-track");
    return { kids: t.children.length, w: Math.round(t.getBoundingClientRect().width) };
  });
  await page.hover(".ticker-viewport");
  await page.waitForTimeout(400);
  const during = await page.evaluate(() => getComputedStyle(document.querySelector(".ticker-track")).animationPlayState);
  await page.mouse.move(1300, 700);
  await page.waitForTimeout(400);
  const after = await page.evaluate(() => getComputedStyle(document.querySelector(".ticker-track")).animationPlayState);
  const drawn = await page.evaluate(() => [...document.querySelectorAll(".sparkline-container svg polyline")].filter((p) => (p.getAttribute("points") || "").length > 10).length);
  console.log("ticker:", JSON.stringify(before), "pause-on-hover:", during, "resume:", after, "sparks:", drawn);
  console.log("errs:", errs.length ? errs.join(" | ") : "(none)");
  await page.screenshot({ path: "C:\\Users\\LINDUC~1\\AppData\\Local\\Temp\\opencode\\verify-saham-1366.png" });
  await browser.close();
})();
