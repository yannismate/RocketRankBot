package db

import (
	"context"
	"encoding/json"
	"time"
)

func (c *cacheDB) SetCachedCommand(ctx context.Context, channelID string, commandName string, cachedCmd *CachedCommand, ttl time.Duration) error {
	cacheKey := cachePrefixRanks + ":" + channelID + ":" + commandName

	jsonBytes, err := json.Marshal(cachedCmd)
	if err != nil {
		return err
	}

	c.client.Set(ctx, cacheKey, string(jsonBytes), ttl)
	return nil
}
