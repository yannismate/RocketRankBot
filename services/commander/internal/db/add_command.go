package db

import (
	"context"
)

func (m *mainDB) AddCommand(ctx context.Context, cmd *BotCommand) error {
	res, err := m.dbPool.Query(ctx, "insert into "+
		"bot_commands "+
		"(command_name, command_cooldown_seconds, message_format, "+
		"twitch_user_id, twitch_response_type, rl_platform, rl_username) "+
		"values "+
		"($1, $2, $3, $4, $5, $6, $7);",
		cmd.CommandName, cmd.CommandCooldownSeconds, cmd.MessageFormat,
		cmd.TwitchUserID, cmd.TwitchResponseType, cmd.RLPlatform, cmd.RLUsername)

	if err == nil {
		res.Close()
	}

	return err
}
