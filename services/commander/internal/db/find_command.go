package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (m *mainDB) FindCommand(ctx context.Context, channelID string, commandName string) (*BotCommand, bool, error) {
	bc := BotCommand{}

	err := m.dbPool.QueryRow(ctx, "select "+
		"command_name, command_cooldown_seconds, message_format, "+
		"twitch_user_id, twitch_response_type, rl_platform, rl_username "+
		"from bot_commands "+
		"where "+
		"twitch_user_id = $1 and command_name = $2;",
		channelID, commandName).Scan(&bc.CommandName, &bc.CommandCooldownSeconds, &bc.MessageFormat, &bc.TwitchUserID,
		&bc.TwitchResponseType, &bc.RLPlatform, &bc.RLUsername)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &bc, true, nil
}
