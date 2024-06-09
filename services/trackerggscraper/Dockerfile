FROM node:22-alpine AS build-env
COPY . /app
WORKDIR /app

RUN npm ci
RUN npm run build
RUN npx puppeteer browsers install chrome
CMD ["out/src/main.js"]