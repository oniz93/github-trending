const express = require('express');
const puppeteer = require('puppeteer');
const axios = require('axios');
const fs = require('fs');
const path = require('path');

const app = express();
const port = 3000;

const API_SERVER_URL = 'http://api-server:8080';
const LOGO_PATH = './assets/logo.png';

app.get('/', async (req, res) => {
    const { id } = req.query;
    if (!id) {
        return res.status(400).send('Repository ID is required');
    }

    try {
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
        const browser = await puppeteer.launch({
            executablePath: process.env.PUPPETEER_EXECUTABLE_PATH,
            args: ['--no-sandbox', '--disable-setuid-sandbox'],
        });
        const page = await browser.newPage();
        await page.setViewport({ width: 1200, height: 630 });
        await page.setContent(html, { waitUntil: 'networkidle0' });
        const screenshot = await page.screenshot({ type: 'png' });
        await browser.close();

        // 4. Return Image
        res.setHeader('Content-Type', 'image/png');
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
