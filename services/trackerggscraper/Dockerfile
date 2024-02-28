FROM node:20 AS build-env
COPY . /app
WORKDIR /app

RUN npm ci --omit=dev
RUN npm run build

FROM gcr.io/distroless/nodejs20-debian12
COPY --from=build-env /app/out /app
WORKDIR /app
CMD ["src/main.js"]