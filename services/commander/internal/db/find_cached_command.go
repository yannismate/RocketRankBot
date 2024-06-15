package db

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
)

func (c *cacheDB) FindCachedCommand(ctx context.Context, channelID string, commandName string) (*CachedCommand, bool, error) {
	cachedString, err := c.client.Get(ctx, cachePrefixCommands+":"+channelID+":"+commandName).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	cc := CachedCommand{}

	err = json.Unmarshal([]byte(cachedString), &cc)
	if err != nil {
		return nil, false, err
	}

	return &cc, true, nil
}
