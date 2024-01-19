package db

import (
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"context"
	"encoding/json"
	"time"
)

func (c *cacheDB) SetCachedRank(ctx context.Context, platform RLPlatform, identifier string, res *trackerggscraper.PlayerCurrentRanksRes, ttl time.Duration) error {
	cacheKey := cachePrefixRanks + ":" + string(platform) + ":" + identifier

	jsonBytes, err := json.Marshal(res)
	if err != nil {
		return err
	}

	c.client.Set(ctx, cacheKey, string(jsonBytes), ttl)
	return nil
}
