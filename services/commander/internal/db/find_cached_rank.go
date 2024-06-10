package db

import (
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
)

func (c *cacheDB) FindCachedRank(ctx context.Context, platform RLPlatform, identifier string) (*trackerggscraper.PlayerCurrentRanksRes, bool, error) {
	cachedString, err := c.client.Get(ctx, cachePrefixRanks+":"+string(platform)+":"+identifier).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	cc := trackerggscraper.PlayerCurrentRanksRes{}

	err = json.Unmarshal([]byte(cachedString), &cc)
	if err != nil {
		return nil, false, err
	}

	return &cc, true, nil
}
