import * as promClient from "prom-client";
import express from "express";
import {register} from "prom-client";
import * as cfg from "../../config.json";
import {logger} from "./logger";
import {rateLimiter} from "./ratelimiting";

promClient.collectDefaultMetrics();
const metricsServer = express();
metricsServer.get('/metrics', async (req, res) => {
    res.contentType(register.contentType).end(await register.metrics());
});
metricsServer.get('/health', async (req, res) => {
   res.send('OK');
});
metricsServer.get('/ready', async (req, res) => {
    res.send('OK');
});

new promClient.Gauge({
    name: "scraper_ratelimit_wait_seconds",
    help: "The scrapers current rate limit wait time",
    async collect() {
        this.set(rateLimiter.secondsUntilNextTry())
    }
});

export const metricResponseTime = new promClient.Histogram({
    name: 'scraper_response_time',
    help: 'Response time in ms',
    labelNames: ["responseCode"],
    buckets: [0.1, 0.5, 1, 2, 5, 10, 30],
});

export const metricCounterRequestCount = new promClient.Counter({
    name: "scraper_incoming_requests",
    labelNames: ["platform"],
    help: "Incoming request count per platform"
});

export function startMetricsServer() {
    (async () => {
        metricsServer.listen(cfg.ADMIN_PORT, () => {
            logger.info({ msg: `Metrics server listening on ${cfg.ADMIN_PORT}` });
        });
    })();
}