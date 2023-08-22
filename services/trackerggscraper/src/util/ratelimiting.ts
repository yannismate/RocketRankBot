import {TrackerGgError, TrackerGgScraper} from "../scraper";
import {logger} from "./logger";

const BLOCKED_WAIT_MIN = 30;
const BLOCKED_WAIT_MAX = 60*8;

export class RateLimiter {

    private blocked: boolean = false;
    private nextRequest: number = Date.now();

    shouldRequest(): boolean {
        return !this.blocked;
    }

    secondsUntilNextTry(): number {
        return Math.ceil(Math.max(0, this.nextRequest - Date.now()) / 1000);
    }

    asyncRetryUntilUnblocked(scraper: TrackerGgScraper, platform: string, user: string, waitSeconds: number = BLOCKED_WAIT_MIN) {
        this.blocked = true;
        this.nextRequest = Date.now() + (waitSeconds * 1000);

        logger.info({ msg: "Rate limit by Cloudflare detected, waiting before retry.", rateLimitWaitSeconds: waitSeconds });

        setTimeout(async () => {
            if (await scraper.fetchRankData(platform, user) != TrackerGgError.CLOUDFLARE_BLOCK) {
                this.blocked = false;
                logger.info({ msg: "Cloudflare rate limit resolved." });
            } else {
                this.asyncRetryUntilUnblocked(scraper, platform, user, Math.max(waitSeconds * 2, BLOCKED_WAIT_MAX));
            }
        }, waitSeconds * 1000);
    }

}

export const rateLimiter = new RateLimiter();