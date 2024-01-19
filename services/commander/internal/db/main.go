package db

import (
	"RocketRankBot/services/commander/internal/config"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type mainDB struct {
	dbPool *pgxpool.Pool
}

type MainDB interface {
	FindCommand(ctx context.Context, channelID string, commandName string) (*BotCommand, bool, error)
	FindUser(ctx context.Context, twitchUserID string) (*BotUser, bool, error)
	AddUser(ctx context.Context, user *BotUser) error
	AddCommand(ctx context.Context, cmd *BotCommand) error
	UpdateUserLogin(ctx context.Context, twitchUserID string, twitchLogin string) error
	DeleteUserData(ctx context.Context, twitchUserID string) error
}

func NewMainDB(cfg *config.CommanderConfig) (MainDB, error) {
	dbPool, err := pgxpool.New(context.Background(), cfg.DB.Main)
	if err != nil {
		zap.L().Error("Error creating Postgres pool", zap.Error(err))
		return nil, err
	}

	return &mainDB{dbPool: dbPool}, nil
}
