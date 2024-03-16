import puppeteer, {Browser} from 'puppeteer';
import {PlayerCurrentRanksRes, RankPlaylist} from "./protos/trackerggscraper.pb";
import {logger} from "./util/logger";

export enum TrackerGgError {
    PLAYER_NOT_FOUND,
    CLOUDFLARE_BLOCK,
    PARSING_ERROR,
    UNKNOWN_ERROR
}
type TrackerGgResult = PlayerCurrentRanksRes | TrackerGgError;

const buildUrl = (platform: string, user: string) => {
    return `https://api.tracker.gg/api/v2/rocket-league/standard/profile/${platform}/${encodeURIComponent(user)}`;
}

export class TrackerGgScraper {

    private browser: Browser | undefined;
    private userAgent: string = "";

    private async start() {
        this.browser = await puppeteer.launch({headless: true});
        this.userAgent = (await this.browser.userAgent()).replace("Headless", "");
    }

    async fetchRankData(platform: string, user: string) : Promise<TrackerGgResult> {
        const text = await this.fetchRankPageText(platform, user);
        if (text.includes("You are being rate limited")) {
            return TrackerGgError.CLOUDFLARE_BLOCK;
        }
        if (text.includes("CollectorResultStatus::NotFound")) {
            return TrackerGgError.PLAYER_NOT_FOUND;
        }

        let parsedResponse: TrackerGgApiResponse;
        try {
            parsedResponse = JSON.parse(text);
        } catch (err) {
            logger.warn({ msg: "Could not parse response JSON", error: err })
            return TrackerGgError.PARSING_ERROR;
        }

        let responseObj: PlayerCurrentRanksRes;
        try {
            responseObj = {
                displayName: parsedResponse.data.platformInfo.platformUserHandle,
                    ranks: parsedResponse.data.segments
                .filter(s => s.type == "playlist" && s.attributes.playlistId != undefined)
                .map(s => {
                    const playlist = playlistMapping.get(s.attributes.playlistId!);
                    if (playlist == undefined) {
                        throw Error(`unknown playlist ${s.attributes.playlistId}`)
                    }

                    return {
                        playlist: playlist,
                        mmr: s.stats.rating?.value || 0,
                        rank: s.stats.tier?.value || 0,
                        division: s.stats.division?.value || 0
                    }
                })
            }
        } catch (err) {
            logger.error({ msg: "Error during tracker.gg response parsing", error: err, parsed_response: parsedResponse });
            return TrackerGgError.UNKNOWN_ERROR;
        }
        return responseObj
    }

    private async fetchRankPageText(platform: string, user: string): Promise<string> {
        if (this.browser == undefined) {
            await this.start();
        }
        if (this.browser == undefined) {
            throw Error("browser not available");
        }
        const page = await this.browser.newPage();

        let content = "";

        try {
            await page.setUserAgent(this.userAgent);
            await page.setExtraHTTPHeaders({
                'Origin': 'https://rocketleague.tracker.network',
                'Referer': 'https://rocketleague.tracker.network/'
            });

            await page.goto(buildUrl(platform, user), { waitUntil: "domcontentloaded" });

            content = await page.evaluate(() =>  {
                return document.querySelector("body")?.innerText;
            }) || "";
        } finally {
            await page.close();
        }

        return content;
    }

}

export const scraper = new TrackerGgScraper();

interface TrackerGgApiResponse {
    data: {
        platformInfo: {
            platformUserHandle: string
        },
        segments: [{
            type: string,
            attributes: {
                playlistId?: number
            },
            stats: {
                tier?: {
                    value: number
                },
                division?: {
                    value: number
                },
                rating?: {
                    value: number
                }
            }
        }]
    }
}

const playlistMapping = new Map<number, RankPlaylist>([
    [0, RankPlaylist.UNRANKED],
    [10, RankPlaylist.RANKED_1V1],
    [11, RankPlaylist.RANKED_2V2],
    [13, RankPlaylist.RANKED_3V3],
    [27, RankPlaylist.HOOPS],
    [28, RankPlaylist.RUMBLE],
    [29, RankPlaylist.DROPSHOT],
    [30, RankPlaylist.SNOWDAY],
    [34, RankPlaylist.TOURNAMENTS]
]);
