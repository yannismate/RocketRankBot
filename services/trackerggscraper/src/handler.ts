import {
    createTrackerGgScraper,
    PlayerCurrentRanksReq,
    PlayerCurrentRanksRes,
    TrackerGgScraper
} from "./protos/trackerggscraper.pb";
import {scraper, TrackerGgError} from "./scraper";
import {rateLimiter} from "./util/ratelimiting";
import {TwirpError} from "twirpscript";
import {metricCounterRequestCount} from "./util/metrics";
import {logger} from "./util/logger";

const trackerGgScraper: TrackerGgScraper = {

    async PlayerCurrentRanks(playerCurrentRanksReq: PlayerCurrentRanksReq): Promise<PlayerCurrentRanksRes> {
        metricCounterRequestCount.labels({ platform: playerCurrentRanksReq.platform }).inc(1);
        if(!rateLimiter.shouldRequest()) {
            throw new TwirpError({
                code: "resource_exhausted",
                msg: "Rate limited by Cloudflare",
                meta: {
                    "secondsUntilNextTry": rateLimiter.secondsUntilNextTry().toString(10)
                }
            });
        }
        const response = await scraper.fetchRankData(playerCurrentRanksReq.platform.toLowerCase(), playerCurrentRanksReq.identifier);
        if (response == TrackerGgError.UNKNOWN_ERROR) {
            throw new TwirpError({
                code: "unknown",
                msg: "Unknown error"
            });
        }
        if (response == TrackerGgError.PARSING_ERROR) {
            throw new TwirpError({
                code: "unknown",
                msg: "Error parsing tracker.gg response"
            });
        }
        if (response == TrackerGgError.PLAYER_NOT_FOUND) {
            throw new TwirpError({
                code: "not_found",
                msg: "Player not found"
            });
        }
        if (response == TrackerGgError.CLOUDFLARE_BLOCK) {
            rateLimiter.asyncRetryUntilUnblocked(scraper, playerCurrentRanksReq.platform, playerCurrentRanksReq.identifier);
            throw new TwirpError({
                code: "resource_exhausted",
                msg: "Rate limited by Cloudflare",
                meta: {
                    "secondsUntilNextTry": rateLimiter.secondsUntilNextTry().toString(10)
                }
            });
        }
        return Promise.resolve(response as PlayerCurrentRanksRes);
    }

}

export const trackerGgScraperHandler = createTrackerGgScraper(trackerGgScraper);