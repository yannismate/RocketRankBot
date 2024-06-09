package db

import "context"

func (m *mainDB) UpdateCommand(ctx context.Context, cmd *BotCommand) error {
	res, err := m.dbPool.Query(ctx, "update "+
		"bot_commands "+
		"set "+
		"(command_cooldown_seconds, message_format, twitch_response_type, rl_platform, rl_username) = ($1, $2, $3, $4, $5) "+
		"where "+
		"twitch_user_id = $6 "+
		"and command_name = $7;",
		cmd.CommandCooldownSeconds, cmd.MessageFormat, cmd.TwitchResponseType,
		cmd.RLPlatform, cmd.RLUsername, cmd.TwitchUserID, cmd.CommandName)

	if err == nil {
		res.Close()
	}

	return err
}
