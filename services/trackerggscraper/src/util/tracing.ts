import {ServerRequest, TwirpContext} from "twirpscript/dist/runtime/server";
import { Guid } from 'guid-ts';
import { logger } from './logger'
import {TwirpError} from "twirpscript";
import {metricResponseTime} from "./metrics";

export interface Context {
    tracing: {
        "trace-id": string,
        "span-id": string,
        "parent-span-id": string
    },
    timer: (labels?: Partial<Record<any, any>>) => number
}

export interface TwirpResponse {
    statusCode: number
}

export const tracingRequestReceived = async (ctx: TwirpContext<Context>, req: ServerRequest) => {
    ctx.timer = metricResponseTime.startTimer();
    ctx.tracing = {
        "trace-id": req.headers["trace-id"] as string | undefined || Guid.newGuid().toString(),
        "parent-span-id": req.headers["span-id"] as string | undefined || "",
        "span-id": Guid.newGuid().toString()
    };
    logger.trace({...{
        request_method: req.method,
        request_url: req.url,
        request_headers: req.headers
    }, ...ctx.tracing});
}

export const tracingResponseSent = async (ctx: TwirpContext<Context>, res: TwirpResponse) => {
    logger.trace({...{
        response_status: res.statusCode
    }, ...ctx.tracing});
    ctx.timer({ responseCode: res.statusCode });
}

export const tracingError = async (ctx: TwirpContext<Context>, error: TwirpError) => {
    logger.warn({
        tracing: ctx.tracing,
        error: error
    });
}