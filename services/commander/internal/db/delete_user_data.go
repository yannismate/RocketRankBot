package db

import "context"

func (m *mainDB) DeleteUserData(ctx context.Context, twitchUserID string) error {
	rows, err := m.dbPool.Query(ctx, "delete from "+
		"event_sub_subscriptions "+
		"where "+
		"twitch_user_id = $1;", twitchUserID)
	if err != nil {
		return err
	}
	rows.Close()

	rows, err = m.dbPool.Query(ctx, "delete from "+
		"bot_commands "+
		"where "+
		"twitch_user_id = $1;", twitchUserID)
	if err != nil {
		return err
	}
	rows.Close()

	rows, err = m.dbPool.Query(ctx, "delete from "+
		"bot_users "+
		"where "+
		"twitch_user_id = $1;", twitchUserID)
	if err == nil {
		rows.Close()
	}

	return err
}
