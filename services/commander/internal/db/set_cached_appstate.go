package db

import (
	"context"
	"encoding/json"
)

func (c *cacheDB) SetCachedAppState(ctx context.Context, cachedAppState *CachedAppState) error {
	jsonBytes, err := json.Marshal(cachedAppState)
	if err != nil {
		return err
	}

	c.client.Set(ctx, cacheKeyAppState, string(jsonBytes), 0)
	return nil
}
