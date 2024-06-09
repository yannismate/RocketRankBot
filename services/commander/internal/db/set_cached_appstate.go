package db

import (
	"context"
	"encoding/json"
)

func (c *cacheDB) SetCachedAppState(ctx context.Context, cachedAppState *CachedAppState) error {
	cacheKey := cacheKeyAppState

	jsonBytes, err := json.Marshal(cachedAppState)
	if err != nil {
		return err
	}

	c.client.Set(ctx, cacheKey, string(jsonBytes), 0)
	return nil
}
