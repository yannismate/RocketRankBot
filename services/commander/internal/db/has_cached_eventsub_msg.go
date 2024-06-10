package db

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
)

func (c *cacheDB) HasCachedEventSubMsg(ctx context.Context, messageID string) (bool, error) {
	cacheKey := cachePrefixEventSubMsg + ":" + messageID
	err := c.client.Get(ctx, cacheKey).Err()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
