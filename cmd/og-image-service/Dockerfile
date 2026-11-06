FROM node:26-slim

# No chromium here any more: rendering is done by the obscura service over CDP.
# That alone removed ~1.1 GB from this image and the per-request browser leak.

WORKDIR /app

COPY cmd/og-image-service/package*.json ./
RUN npm install --omit=dev && npm cache clean --force

COPY cmd/og-image-service/. .
COPY web/assets/logo.png ./assets/logo.png

ENV NODE_ENV=production
ENV OBSCURA_CDP_URL=http://obscura:9222

EXPOSE 3000
CMD ["node", "index.js"]
