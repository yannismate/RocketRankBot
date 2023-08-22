import { Logger } from "tslog";

const devMode = process.env["LOG_MODE"] == "dev";

export const logger = new Logger({
    type: devMode ? "pretty" : "json",
    name: "trackerggscraper",
    minLevel: 1,
    hideLogPositionForProduction: !devMode
});