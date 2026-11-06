const express = require('express');
const puppeteer = require('puppeteer-core');
const axios = require('axios');
const fs = require('fs');
const path = require('path');

const app = express();
const port = 3000;

const API_SERVER_URL = process.env.API_SERVER_URL || 'http://api-server:8080';
const LOGO_PATH = './assets/logo.png';

// --- Rendering: one long-lived Obscura instance over CDP --------------------
// The old code did puppeteer.launch() per request: ~1.3 s of launch inside a
// ~3.4 s render, and a burst left orphaned browsers behind (895 MiB of them).
// Obscura renders this card in ~0.2 s warm and its output is byte-stable
// across platforms, because it ships its own fonts.
const CDP_URL = (process.env.OBSCURA_CDP_URL || 'http://obscura:9222').replace(/\/$/, '');
// Obscura refuses a non-loopback bind without a >=32-byte bearer token; the same
// token must ride the HTTP discovery call and the WebSocket handshake.
const CDP_TOKEN = process.env.OBSCURA_CDP_TOKEN || '';
function cdpHeaders() {
    return CDP_TOKEN ? { Authorization: `Bearer ${CDP_TOKEN}` } : {};
}
const VIEWPORT = { width: 1200, height: 630 };
const RENDER_CONCURRENCY = Number(process.env.OG_RENDER_CONCURRENCY || 4);
const CACHE_TTL_MS = Number(process.env.OG_CACHE_TTL_MS || 10 * 60 * 1000);
const CACHE_MAX_ENTRIES = Number(process.env.OG_CACHE_MAX || 256);

let browserPromise = null;
let connectingPromise = null;
function isBrowserAlive(browser) {
    // puppeteer-core >= 23 dropped Browser.isConnected() in favour of the
    // `connected` getter; support both so an upgrade cannot break rendering.
    return typeof browser.isConnected === 'function' ? browser.isConnected() : Boolean(browser.connected);
}

async function resolveWsEndpoint() {
    // obscura may still be starting up; retry for ~30 s before giving up
    for (let attempt = 0; attempt < 60; attempt++) {
        try {
            const res = await axios.get(`${CDP_URL}/json/version`, { timeout: 2000, headers: cdpHeaders() });
            if (res.data && res.data.webSocketDebuggerUrl) {
                // obscura advertises ws://127.0.0.1:9222/... regardless of --host, so
                // rewrite the authority to the host we actually reach it on.
                const base = new URL(CDP_URL);
                const ws = new URL(res.data.webSocketDebuggerUrl);
                ws.hostname = base.hostname;
                ws.port = base.port || ws.port;
                return ws.toString();
            }
        } catch (_) { /* not up yet */ }
        await new Promise(r => setTimeout(r, 500));
    }
    throw new Error(`Obscura CDP endpoint not reachable at ${CDP_URL}`);
}

async function getBrowser() {
    if (browserPromise) {
        try {
            const browser = await browserPromise;
            if (isBrowserAlive(browser)) return browser;
        } catch (_) { /* reconnect below */ }
        browserPromise = null;
    }
    if (!connectingPromise) {
        connectingPromise = (async () => {
            const ws = await resolveWsEndpoint();
            const browser = await puppeteer.connect({
                browserWSEndpoint: ws,
                headers: cdpHeaders(),
                defaultViewport: null,
            });
            browser.on('disconnected', () => { browserPromise = null; });
            return browser;
        })().finally(() => { connectingPromise = null; });
    }
    browserPromise = connectingPromise;
    return browserPromise;
}

// Bounds how many pages a crawler burst can open at once (Twitterbot sent 11).
let activeRenders = 0;
const renderQueue = [];
async function acquireSlot() {
    if (activeRenders >= RENDER_CONCURRENCY) {
        await new Promise(resolve => renderQueue.push(resolve));
    }
    activeRenders++;
}
function releaseSlot() {
    activeRenders--;
    const next = renderQueue.shift();
    if (next) next();
}

async function renderCard(html) {
    const browser = await getBrowser();
    let page = null;
    try {
        page = await browser.newPage();
        await page.setViewport(VIEWPORT);
        // Obscura's CDP does not implement Page.setDocumentContent, so
        // page.setContent() fails with "Unknown Page method: setDocumentContent".
        // Navigating to a data: URL produces the identical image.
        const dataUrl = 'data:text/html;base64,' + Buffer.from(html, 'utf-8').toString('base64');
        await page.goto(dataUrl, { waitUntil: 'load', timeout: 20000 });
        // puppeteer-core 25 returns a Uint8Array, which Express would serialise as
        // JSON ({0:137,1:80,...}) instead of sending a PNG. Normalise to a Buffer.
        return Buffer.from(await page.screenshot({ type: 'png' }));
    } finally {
        if (page) { try { await page.close(); } catch (_) { /* already gone */ } }
    }
}

const cardCache = new Map();   // id -> { png, at }
function cacheGet(id) {
    const hit = cardCache.get(id);
    if (!hit) return null;
    if (Date.now() - hit.at > CACHE_TTL_MS) { cardCache.delete(id); return null; }
    return hit.png;
}
function cacheSet(id, png) {
    if (cardCache.size >= CACHE_MAX_ENTRIES) {
        const oldest = cardCache.keys().next().value;
        if (oldest !== undefined) cardCache.delete(oldest);
    }
    cardCache.set(id, { png, at: Date.now() });
}

app.get('/health', async (req, res) => {
    try {
        const browser = await getBrowser();
        if (!isBrowserAlive(browser)) throw new Error('CDP connection lost');
        res.json({ ok: true, cached: cardCache.size, renders: activeRenders });
    } catch (error) {
        res.status(503).json({ ok: false, error: error.message });
    }
});

app.get('/', async (req, res) => {
    const { id } = req.query;
    if (!id) {
        return res.status(400).send('Repository ID is required');
    }

    try {
        const cached = cacheGet(id);
        if (cached) {
            res.setHeader('Content-Type', 'image/png');
            res.setHeader('X-OG-Cache', 'hit');
            return res.send(cached);
        }

        // 1. Fetch Data
        const repoDetailsResponse = await axios.get(`${API_SERVER_URL}/repository/${id}`);
        const { repository } = repoDetailsResponse.data;

        let readmeHtml = '';
        try {
            const readmeResponse = await axios.get(`${API_SERVER_URL}/getReadme?repoId=${id}`);
            readmeHtml = readmeResponse.data;
        } catch (error) {
            console.error(`Could not fetch README for repo ${id}:`, error.message);
        }

        // 2. Render HTML
        const html = getHtml(repository, readmeHtml);

        // 3. Generate Image
        let screenshot;
        await acquireSlot();
        try {
            screenshot = await renderCard(html);
        } finally {
            releaseSlot();
        }
        cacheSet(id, screenshot);

        // 4. Return Image
        res.setHeader('Content-Type', 'image/png');
        res.setHeader('X-OG-Cache', 'miss');
        res.send(screenshot);

    } catch (error) {
        console.error(`Failed to generate OG image for repo ${id}:`, error);
        res.status(500).send('Failed to generate OG image');
    }
});

function getHtml(repository, readmeHtml) {
    const logoBase64 = fs.readFileSync(LOGO_PATH, 'base64');
    const logoSrc = `data:image/png;base64,${logoBase64}`;

    const readmeSnippet = readmeHtml
        .replace(/<style[^>]*>.*<\/style>/gs, "")
        .replace(/<script[^>]*>.*<\/script>/gs, "")
        .replace(/<[^>]+>/g, ' ')
        .replace(/\s\s+/g, ' ')
        .trim()
        .substring(0, 400);

    const description = repository.description.Valid ? repository.description.String : '';

    return `
<html>
  <head>
    <style>
      body {
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
        background-color: #0d1117;
        color: white;
        padding: 50px;
        width: 1200px;
        height: 630px;
        box-sizing: border-box;
        display: flex;
        flex-direction: column;
        justify-content: space-between;
      }
      .header {
        display: flex;
        align-items: center;
        padding-bottom: 20px;
        border-bottom: 1px solid #30363d;
      }
      .logo {
        width: 100px;
        height: 100px;
        margin-right: 30px;
        border-radius: 6px;
      }
      .title-section {
        display: flex;
        flex-direction: column;
        overflow: hidden;
      }
      .title {
        font-size: 48px;
        font-weight: 600;
        color: #c9d1d9;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
      .description {
        font-size: 24px;
        color: #8b949e;
        margin-top: 8px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
      .readme {
        font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace;
        font-size: 18px;
        color: #c9d1d9;
        max-height: 250px;
        overflow: hidden;
        text-overflow: ellipsis;
        margin-top: 20px;
        line-height: 1.6;
        -webkit-mask-image: linear-gradient(to bottom, black 50%, transparent 100%);
        mask-image: linear-gradient(to bottom, black 50%, transparent 100%);
      }
      .footer {
        font-size: 20px;
        color: #8b949e;
        text-align: right;
      }
    </style>
  </head>
  <body>
    <div>
        <div class="header">
          <img src="${logoSrc}" class="logo" />
          <div class="title-section">
            <div class="title">${repository.full_name}</div>
            <div class="description">${description}</div>
          </div>
        </div>
        <div class="readme">
          ${readmeSnippet}...
        </div>
    </div>
    <div class="footer">
        app.gitfinder.dev
    </div>
  </body>
</html>
    `;
}

const botUserAgents = [
  'googlebot', 'yahoo! slurp', 'bingbot', 'yandex', 'baiduspider', 'facebookexternalhit',
  'twitterbot', 'rogerbot', 'linkedinbot', 'embedly', 'quora link preview', 'showyoubot',
  'outbrain', 'pinterest/0.', 'pinterestbot', 'slackbot', 'vkshare', 'w3c_validator',
  'redditbot', 'applebot', 'whatsapp', 'flipboard', 'tumblr', 'bitlybot', 'skypeuripreview',
  'nuzzel', 'discordbot', 'google page speed', 'qwantify'
];

app.get('/repository/:id', async (req, res) => {
    const userAgent = req.headers['user-agent'] || '';
    const isBot = botUserAgents.some(bot => userAgent.toLowerCase().includes(bot));
    const indexPath = path.resolve(__dirname, 'index.html');

    if (!fs.existsSync(indexPath)) {
        return res.status(500).send('Failed to render page.');
    }

    if (isBot) {
        console.log(`Bot detected: ${userAgent}. Serving SSR version.`);
        try {
            const { id } = req.params;
            const repoDetailsResponse = await axios.get(`${API_SERVER_URL}/repository/${id}`);
            const { repository } = repoDetailsResponse.data;

            let indexHtml = fs.readFileSync(indexPath, 'utf-8');

            const title = `${repository.full_name} - GitFinder`;
            const description = repository.description.Valid ? repository.description.String : '';
            const ogImage = `https://api.gitfinder.dev/api/og?id=${id}`;

            const metaTags = `
<title>${title}</title>
<meta name="description" content="${description}">
<meta property="og:title" content="${repository.full_name}">
<meta property="og:description" content="${description}">
<meta property="og:image" content="${ogImage}">
<meta property="twitter:card" content="summary_large_image">
<meta property="twitter:title" content="${repository.full_name}">
<meta property="twitter:description" content="${description}">
<meta property="twitter:image" content="${ogImage}">
`;
            // Using a more robust replacement
            indexHtml = indexHtml.replace(/<title>.*?<\/title>/, metaTags);

            res.setHeader('Content-Type', 'text/html');
            return res.send(indexHtml);
        } catch (error) {
            console.error(`Failed to SSR for bot, repo ${req.params.id}:`, error.message);
            return res.sendFile(indexPath); // Fallback for errors
        }
    } else {
        // For regular users, serve the standard SPA
        return res.sendFile(indexPath);
    }
});

app.listen(port, () => {
    console.log(`OG Image service listening on port ${port}`);
});
