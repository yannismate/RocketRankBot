package db

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
)

func (c *cacheDB) GetCachedAppState(ctx context.Context) (*CachedAppState, bool, error) {
	cachedString, err := c.client.Get(ctx, cacheKeyAppState).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	cas := CachedAppState{}

	err = json.Unmarshal([]byte(cachedString), &cas)
	if err != nil {
		return nil, false, err
	}

	return &cas, true, nil
}
