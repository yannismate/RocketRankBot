package db

import (
	"RocketRankBot/services/commander/internal/config"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type mainDB struct {
	dbPool *pgxpool.Pool
}

type MainDB interface {
	FindCommand(ctx context.Context, channelID string, commandName string) (*BotCommand, bool, error)
	FindUser(ctx context.Context, twitchUserID string) (*BotUser, bool, error)
	FindAllTwitchLogins(ctx context.Context) (*[]string, error)
	AddUser(ctx context.Context, user *BotUser) error
	AddCommand(ctx context.Context, cmd *BotCommand) error
	DeleteCommand(ctx context.Context, channelId string, commandName string) error
	UpdateUserLogin(ctx context.Context, twitchUserID string, twitchLogin string) error
	DeleteUserData(ctx context.Context, twitchUserID string) error
}

func NewMainDB(cfg *config.CommanderConfig) (MainDB, error) {
	dbPool, err := pgxpool.New(context.Background(), cfg.DB.Main)
	if err != nil {
		log.Err(err).Msg("Error creating postgres pool")
		return nil, err
	}

	return &mainDB{dbPool: dbPool}, nil
}
