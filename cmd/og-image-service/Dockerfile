FROM node:18-slim

ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true

RUN apt-get update \
    && apt-get install -yq chromium \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY cmd/og-image-service/package*.json ./
RUN npm install

COPY cmd/og-image-service/. .
COPY web/assets/logo.png ./assets/logo.png

ENV PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium

EXPOSE 3000
CMD ["node", "index.js"]
