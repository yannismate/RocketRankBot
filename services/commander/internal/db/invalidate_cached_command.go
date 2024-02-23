package db

import "context"

func (c *cacheDB) InvalidateCachedCommand(ctx context.Context, channelID string, commandName string) error {
	cacheKey := cachePrefixRanks + ":" + channelID + ":" + commandName
	res := c.client.Del(ctx, cacheKey)
	return res.Err()
}
