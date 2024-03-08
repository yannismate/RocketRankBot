package db

import (
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"time"
)

const (
	cachePrefixCommands = "command"
	cachePrefixRanks    = "rank"
)

type cacheDB struct {
	client   *redis.Client
	lastPing time.Time
}

type CacheDB interface {
	IsConnected() bool
	FindCachedCommand(ctx context.Context, channelID string, commandName string) (*CachedCommand, bool, error)
	FindCachedRank(ctx context.Context, platform RLPlatform, identifier string) (*trackerggscraper.PlayerCurrentRanksRes, bool, error)
	SetCachedCommand(ctx context.Context, channelID string, commandName string, cachedCmd *CachedCommand, ttl time.Duration) error
	SetCachedRank(ctx context.Context, platform RLPlatform, identifier string, res *trackerggscraper.PlayerCurrentRanksRes, ttl time.Duration) error
	InvalidateCachedCommand(ctx context.Context, channelID string, commandName string) error
}

func NewCache(cfg *config.CommanderConfig) (CacheDB, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.DB.Cache,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res := client.Ping(ctx)
	if res.Err() != nil {
		return nil, res.Err()
	}
	return &cacheDB{
		client:   client,
		lastPing: time.Now(),
	}, nil
}

func (c *cacheDB) IsConnected() bool {
	if c.lastPing.Add(time.Second).After(time.Now()) {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := c.client.Ping(ctx).Err()

	if err == nil {
		c.lastPing = time.Now()
		return true
	}
	log.Warn().Err(err).Msg("Cache database failed to respond to ping")
	return false
}
