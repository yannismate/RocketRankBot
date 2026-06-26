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
import {registerShutdown} from "./shutdown";

startMetricsServer();

const services = [trackerGgScraperHandler];
const app = createTwirpServer<CustomTwirpContext, typeof services>(services, { debug: false });
app.on("requestReceived", tracingRequestReceived);
app.on("responseSent", tracingResponseSent);
app.on("error", tracingError);

const server = createServer(app);
registerShutdown((exitCode: number) => {
    logger.info({ msg: `Stopping HTTP server due to error...` });
    server.close(() => {
        logger.info({ msg: `Server closed, exiting.` });
        process.exitCode = exitCode;
    });
    setTimeout(() => {
        logger.error({ msg: `Shutdown failed, forcing process exit.` });
        process.exit(exitCode);
    }, 10000).unref();
});

server.listen(cfg.RPC_PORT, () => {
    logger.info({ msg: `App listening for twirp requests on :${cfg.RPC_PORT}` });
});