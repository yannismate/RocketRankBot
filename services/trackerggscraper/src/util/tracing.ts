import {ServerRequest, TwirpContext} from "twirpscript/dist/runtime/server";
import { Guid } from 'guid-ts';
import { logger } from './logger'
import {TwirpError} from "twirpscript";
import {metricResponseTime} from "./metrics";

export interface Context {
    tracing: {
        traceId: string,
        spanId: string,
        parentSpanId: string
    },
    timer: (labels?: Partial<Record<any, any>>) => number
}

export interface TwirpResponse {
    statusCode: number
}

export const tracingRequestReceived = async (ctx: TwirpContext<Context>, req: ServerRequest) => {
    ctx.timer = metricResponseTime.startTimer();
    ctx.tracing = {
        traceId: req.headers["trace-id"] as string | undefined || Guid.newGuid().toString(),
        parentSpanId: req.headers["span-id"] as string | undefined || "",
        spanId: Guid.newGuid().toString()
    };
    logger.trace({
        tracing: ctx.tracing,
        request_method: req.method,
        request_url: req.url,
        request_headers: req.headers
    });
}

export const tracingResponseSent = async (ctx: TwirpContext<Context>, res: TwirpResponse) => {
    logger.trace({
        tracing: ctx.tracing,
        response_status: res.statusCode
    });
    ctx.timer({ responseCode: res.statusCode });
}

export const tracingError = async (ctx: TwirpContext<Context>, error: TwirpError) => {
    logger.trace({
        tracing: ctx.tracing,
        error: error
    });
}