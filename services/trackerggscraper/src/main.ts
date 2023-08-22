import * as cfg from '../config.json';
import { createServer } from "node:http";
import { createTwirpServer } from "twirpscript";
import { trackerGgScraperHandler } from "./handler";
import {startMetricsServer} from "./util/metrics";
import {
    Context as CustomTwirpContext,
    tracingError,
    tracingRequestReceived,
    tracingResponseSent
} from "./util/tracing";
import {logger} from "./util/logger";

startMetricsServer();

const services = [trackerGgScraperHandler];
const app = createTwirpServer<CustomTwirpContext, typeof services>(services, { debug: false });
app.on("requestReceived", tracingRequestReceived);
app.on("responseSent", tracingResponseSent);
app.on("error", tracingError);

createServer(app).listen(cfg.APP_PORT, () => {
    logger.info({ msg: `App listening on ${cfg.APP_PORT}` });
});