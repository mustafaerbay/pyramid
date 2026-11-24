// Takes the screenshots the posters are cut from, out of a local Pyramid
// (frontend on :3000, API on :8080) loaded with the demo data.
// PY_EMAIL / PY_PASS: a local test account. CHROME: path to Chrome.
import puppeteer from 'puppeteer-core';
import { mkdirSync } from 'node:fs';

const APP = process.env.PY_APP ?? 'http://localhost:3000';
const API = process.env.PY_API ?? 'http://localhost:8080';
const CHROME = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const OUT = new URL('./shots/', import.meta.url).pathname;
// Which demo records to show.
const PROJECT_KEY = process.env.PY_PROJECT ?? 'CAD';
const PERSON = process.env.PY_PERSON ?? 'Priya';
const DASHBOARD = process.env.PY_DASHBOARD ?? 'Team overview';

if (!process.env.PY_EMAIL || !process.env.PY_PASS) {
	console.error('Set PY_EMAIL and PY_PASS to a local test account.');
	process.exit(1);
}
mkdirSync(OUT, { recursive: true });

const browser = await puppeteer.launch({ executablePath: CHROME, headless: true });
const page = await browser.newPage();
await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });
await page.goto(APP + '/login', { waitUntil: 'networkidle0' });
await page.type('input[type=email]', process.env.PY_EMAIL);
await page.type('input[type=password]', process.env.PY_PASS);
await Promise.all([page.waitForNavigation({ waitUntil: 'networkidle0' }).catch(() => {}), page.click('button[type=submit]')]);
await new Promise((r) => setTimeout(r, 1500));

const get = (path) => page.evaluate((u) => fetch(u, { credentials: 'include' }).then((r) => r.json()), API + path);
const project = (await get('/api/v1/projects?pageSize=100')).data.find((p) => p.key === PROJECT_KEY);
const person = (await get('/api/v1/people?pageSize=100')).data.find((p) => p.firstName === PERSON);
const dashboard = (await get('/api/v1/dashboards?pageSize=100')).data.find((d) => d.name === DASHBOARD);
if (!project || !person || !dashboard) throw new Error('Demo data not found: check PY_PROJECT, PY_PERSON and PY_DASHBOARD.');

const P = `/projects/${project.id}`;
// name, path, viewport width x height, locale
const shots = [
	['kanban', `${P}/board`],
	['sprints', `${P}/sprints`],
	['plans', '/plans'],
	['dash', `/dashboards/${dashboard.id}`],
	['m_person', `/people/${person.id}`, 390, 844]
];
for (const [name, path, w = 1440, h = 900, locale = 'en'] of shots) {
	await page.evaluate((l) => {
		localStorage.setItem('pyramid.locale', l);
		localStorage.setItem('pyramid.taskView', 'board');
	}, locale);
	await page.setViewport({ width: w, height: h, deviceScaleFactor: 2, isMobile: w < 600, hasTouch: w < 600 });
	await page.goto(APP + path, { waitUntil: 'networkidle0' });
	await new Promise((r) => setTimeout(r, 1200));
	await page.screenshot({ path: `${OUT}${name}.png` });
	console.log('shot', name);
}
await browser.close();
