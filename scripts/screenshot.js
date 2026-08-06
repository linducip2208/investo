const { chromium } = require('playwright');

(async () => {
    const browser = await chromium.launch();
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
    
    // Login first
    await page.goto('http://localhost:8000/login', { waitUntil: 'networkidle', timeout: 10000 }).catch(() => {});
    await page.fill('input[name="email"]', 'admin@investo.test');
    await page.fill('input[name="password"]', 'password');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    // Dashboard
    await page.goto('http://localhost:8000/dashboard', { waitUntil: 'networkidle', timeout: 10000 }).catch(() => {});
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'web/static/screens/dashboard.png', fullPage: false });
    console.log('Dashboard OK');
    
    // Stock list
    await page.goto('http://localhost:8000/saham', { waitUntil: 'networkidle', timeout: 10000 }).catch(() => {});
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'web/static/screens/saham.png', fullPage: false });
    console.log('Saham OK');
    
    // Heatmap  
    await page.goto('http://localhost:8000/market/heatmap', { waitUntil: 'networkidle', timeout: 10000 }).catch(() => {});
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'web/static/screens/heatmap.png', fullPage: false });
    console.log('Heatmap OK');
    
    await browser.close();
    console.log('Done');
})();
