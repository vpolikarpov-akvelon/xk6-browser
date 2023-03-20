import { chromium } from 'k6/x/browser';

export default async function () {
  const browser = chromium.launch({headless: true});
  const context = browser.newContext();
  context.addInitScript(`{
    function print(metric) {
      const m = {
        id: metric.id,
        name: metric.name,
        value: metric.value,
        rating: metric.rating,
        delta: metric.delta,
        numEntries: metric.entries.length,
        navigationType: metric.navigationType,
        url: window.location.href,
      }
      console.log('xk6-browser.web.vital.metric=' + JSON.stringify(m))
      window.testbinding(JSON.stringify(m))
    }

    async function load() {
      let {
        onCLS, onFID, onLCP, onFCP, onINP, onTTFB
      } = await import('https://unpkg.com/web-vitals@3?module');

      onCLS(print);
      onFID(print);
      onLCP(print);
  
      onFCP(print);
      onINP(print);
      onTTFB(print);
    }

    load();
  }`);

  try {
    runTest2(context)
    
    var page = context.newPage(); // AddBinding
    var page2 = context.newPage(); // AddBinding
    await runTest(page, 'https://grafana.com', 'https://play.grafana.org/', 'grafana_screenshot.png')
    page.close() // RemoveBinding -- only affects the session it's in
    await runTest(page2, 'https://k6.io', '/jobs/', 'k6_screenshot.png')
    page2.close() // RemoveBinding
  } finally {
    browser.close();
  }
}

async function runTest2(context) {
  const page = context.newPage();
  page.close();
  console.log("runTest2 done");
}

async function runTest(page, url, clickHref, screenshotFile) {
  try{
    await page.goto(url, { waitUntil: 'networkidle' })

    await Promise.all([
      page.waitForNavigation({ waitUntil: 'networkidle' }),
      page.locator('a[href="' + clickHref + '"]').click(),
    ]);

    page.screenshot({ path: screenshotFile });
  } finally {
    console.log("runTest done");
  }
}
