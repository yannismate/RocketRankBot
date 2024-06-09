package db

import (
	"context"
)

func (m *mainDB) DeleteCommand(ctx context.Context, channelId string, commandName string) error {
	res, err := m.dbPool.Query(ctx, "delete from "+
		"bot_commands "+
		"where "+
		"twitch_user_id = $1 "+
		"and command_name = $2;",
		channelId, commandName)

	if err == nil {
		res.Close()
	}

	return err
}
