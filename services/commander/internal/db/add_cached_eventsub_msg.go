package db

import (
	"context"
	"time"
)

func (c *cacheDB) AddCachedEventSubMsg(ctx context.Context, messageID string) error {
	cacheKey := cachePrefixEventSubMsg + ":" + messageID
	return c.client.Set(ctx, cacheKey, "1", time.Minute*11).Err()
}
