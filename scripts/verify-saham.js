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
    await page.waitForTimeout(4000);
    const m = await page.evaluate(() => {
      const de = document.documentElement;
      const aside = document.querySelector("aside.sidebar");
      const track = document.querySelector(".ticker-track");
      const cs = track ? getComputedStyle(track) : null;
      const spark = document.querySelectorAll(".sparkline-container svg polyline[points^='0']");
      const sparkDrawn = [...document.querySelectorAll(".sparkline-container svg polyline")].filter((p) => (p.getAttribute("points") || "").length > 10).length;
      return {
        overflowX: de.scrollWidth - de.clientWidth,
        asideDisplay: aside ? getComputedStyle(aside).display : "?",
        animDur: cs ? cs.animationDuration : "?",
        trackW: track ? Math.round(track.getBoundingClientRect().width) : -1,
        parW: track && track.parentElement ? Math.round(track.parentElement.getBoundingClientRect().width) : -1,
        sparkTotal: document.querySelectorAll(".sparkline-container").length,
        sparkDrawn,
        nTh: document.querySelectorAll("thead th").length,
        nTd: (document.querySelectorAll("tbody tr")[0] || { querySelectorAll: () => [] }).querySelectorAll("td").length,
      };
    });
    // hover-pause check at desktop widths
    let paused = "?";
    if (vp.w >= 1024) {
      paused = await page.evaluate(() => {
        const track = document.querySelector(".ticker-track");
        const par = track.parentElement;
        const r = par.getBoundingClientRect();
        const ev = new MouseEvent("mouseover", { bubbles: true, clientX: r.left + 10, clientY: r.top + 5 });
        par.dispatchEvent(ev);
        return getComputedStyle(track).animationPlayState;
      });
    }
    console.log(`== ${vp.w}x${vp.h} ==`, JSON.stringify(m), "hoverPause:", paused);
    console.log("errs:", errs.length ? errs.join(" | ") : "(none)");
    await page.screenshot({ path: `C:\\Users\\LINDUC~1\\AppData\\Local\\Temp\\opencode\\verify-saham-${vp.w}.png` });
    await ctx.close();
  }
  await browser.close();
})();
