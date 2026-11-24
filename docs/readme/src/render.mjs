// Renders posters.html into the README images (docs/readme/*.png).
import puppeteer from 'puppeteer-core';

const CHROME = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const here = new URL('.', import.meta.url).pathname;
const out = new URL('../', import.meta.url).pathname;

const browser = await puppeteer.launch({ executablePath: CHROME, headless: true, args: ['--allow-file-access-from-files'] });
const page = await browser.newPage();
await page.setViewport({ width: 2400, height: 1200, deviceScaleFactor: 1 });
await page.goto('file://' + here + 'posters.html', { waitUntil: 'networkidle0' });
await page.evaluate(() => document.fonts.ready);
for (const id of ['hero', 'row1', 'row2', 'cta']) {
	await (await page.$('#' + id)).screenshot({ path: `${out}${id}.png` });
	console.log('poster', id);
}
await browser.close();
