const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  for (const vp of [{ w: 1920, h: 1080 }, { w: 1366, h: 768 }, { w: 1024, h: 768 }, { w: 390, h: 844 }]) {
    const ctx = await browser.newContext({ viewport: { width: vp.w, height: vp.h } });
    const page = await ctx.newPage();
    const errs = [];
    page.on("pageerror", (e) => errs.push(String(e && e.message || e).slice(0, 100)));
    await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
    await page.waitForTimeout(2500);
    const m = await page.evaluate(() => {
      const de = document.documentElement;
      const ticker = document.querySelector(".animate-ticker");
      const tpar = ticker ? ticker.parentElement : null;
      const aside = document.querySelector("aside.sidebar");
      const main = document.querySelector("main");
      const table = document.querySelector("table");
      const ths = [...document.querySelectorAll("thead th")].map((th) => Math.round(th.getBoundingClientRect().left));
      const firstRow = document.querySelectorAll("tbody tr")[0];
      const tds = firstRow ? [...firstRow.querySelectorAll("td")].map((td) => Math.round(td.getBoundingClientRect().left)) : [];
      const ar = aside ? aside.getBoundingClientRect() : null;
      const mr = main ? main.getBoundingClientRect() : null;
      return {
        scrollW: de.scrollWidth, clientW: de.clientWidth,
        overflowX: de.scrollWidth - de.clientWidth,
        tickerParH: tpar ? Math.round(tpar.getBoundingClientRect().height) : -1,
        tickerH: ticker ? Math.round(ticker.getBoundingClientRect().height) : -1,
        tickerKids: ticker ? ticker.children.length : -1,
        asideW: ar ? Math.round(ar.width) : -1,
        asideVisible: ar ? (ar.width > 0 && ar.left > -ar.width) : false,
        mainLeft: mr ? Math.round(mr.left) : -1,
        mainW: mr ? Math.round(mr.width) : -1,
        nTh: ths.length, nTd: tds.length, ths, tds,
        tableScrollW: table ? (table.scrollWidth - table.clientWidth) : -1,
      };
    });
    console.log(`== ${vp.w}x${vp.h} ==`);
    console.log(JSON.stringify(m));
    console.log("errs:", errs.length ? errs.join(" | ") : "(none)");
    await page.screenshot({ path: `C:\\Users\\LINDUC~1\\AppData\\Local\\Temp\\opencode\\audit-saham-${vp.w}.png` });
    await ctx.close();
  }
  await browser.close();
})();
