package db

import (
	"RocketRankBot/services/commander/internal/config"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"time"
)

type mainDB struct {
	dbPool   *pgxpool.Pool
	lastPing time.Time
}

type MainDB interface {
	IsConnected() bool
	FindCommand(ctx context.Context, channelID string, commandName string) (*BotCommand, bool, error)
	FindUserCommands(ctx context.Context, channelID string) (*[]BotCommand, error)
	FindUser(ctx context.Context, twitchUserID string) (*BotUser, bool, error)
	AddUser(ctx context.Context, user *BotUser) error
	AddCommand(ctx context.Context, cmd *BotCommand) error
	UpdateCommand(ctx context.Context, cmd *BotCommand) error
	DeleteCommand(ctx context.Context, channelId string, commandName string) error
	DeleteUserData(ctx context.Context, twitchUserID string) error
	FindEventSubSubscriptionsForTwitchUserID(ctx context.Context, twitchUserID string) (*[]EventSubSubscription, error)
	AddEventSubSubscription(ctx context.Context, sub *EventSubSubscription) error
	DeleteEventSubSubscription(ctx context.Context, subscriptionID string) error
	FindEventSubSubscriptionByID(ctx context.Context, eventSubID string) (*EventSubSubscription, bool, error)
	UpdateUserAuthenticationFlag(ctx context.Context, twitchUserID string, isAuthed bool) error
}

func NewMainDB(cfg *config.CommanderConfig) (MainDB, error) {
	dbPool, err := pgxpool.New(context.Background(), cfg.DB.Main)
	if err != nil {
		log.Err(err).Msg("Error creating postgres pool")
		return nil, err
	}

	return &mainDB{
		dbPool:   dbPool,
		lastPing: time.Now(),
	}, nil
}

func (m *mainDB) IsConnected() bool {
	if m.lastPing.Add(time.Second).After(time.Now()) {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := m.dbPool.Ping(ctx)

	if err == nil {
		m.lastPing = time.Now()
		return true
	}
	log.Warn().Err(err).Msg("Main database failed to respond to ping")
	return false
}
