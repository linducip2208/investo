const { chromium } = require("playwright-core");
const BASE = process.env.INVESTO_URL || "http://127.0.0.1:8000";
(async () => {
  const browser = await chromium.launch();
  for (const vp of [{ w: 1366, h: 768 }, { w: 390, h: 844 }]) {
    const ctx = await browser.newContext({ viewport: { width: vp.w, height: vp.h } });
    const page = await ctx.newPage();
    await page.goto(BASE + "/saham", { waitUntil: "networkidle", timeout: 45000 });
    await page.waitForTimeout(2500);
    const m = await page.evaluate(() => {
      const aside = document.querySelector("aside.sidebar");
      const track = document.querySelector(".ticker-track");
      const cs = track ? getComputedStyle(track) : null;
      const ar = aside ? aside.getBoundingClientRect() : null;
      return {
        asideDisplay: aside ? getComputedStyle(aside).display : "?",
        asideLeft: ar ? Math.round(ar.left) : -1,
        trackKids: track ? track.children.length : -1,
        animName: cs ? cs.animationName : "?",
        animDur: cs ? cs.animationDuration : "?",
        trackW: track ? Math.round(track.getBoundingClientRect().width) : -1,
        parW: track && track.parentElement ? Math.round(track.parentElement.getBoundingClientRect().width) : -1,
      };
    });
    console.log(`== ${vp.w} ==`, JSON.stringify(m));
    await ctx.close();
  }
  await browser.close();
})();
