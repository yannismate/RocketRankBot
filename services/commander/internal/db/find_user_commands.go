package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (m *mainDB) FindUserCommands(ctx context.Context, channelID string) (*[]BotCommand, error) {
	bc := BotCommand{}

	rows, err := m.dbPool.Query(ctx, "select "+
		"command_name, command_cooldown_seconds, message_format, "+
		"twitch_user_id, twitch_response_type, rl_platform, rl_username "+
		"from bot_commands "+
		"where "+
		"twitch_user_id = $1;",
		channelID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			rows.Close()
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var commands []BotCommand
	for rows.Next() {
		cmd := BotCommand{}
		err = rows.Scan(&bc.CommandName, &bc.CommandCooldownSeconds, &bc.MessageFormat, &bc.TwitchUserID,
			&bc.TwitchResponseType, &bc.RLPlatform, &bc.RLUsername)
		if err != nil {
			return nil, err
		}
		commands = append(commands, cmd)
	}

	return &commands, nil
}
