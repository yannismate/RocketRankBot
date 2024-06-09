package db

import "context"

func (m *mainDB) DeleteUserData(ctx context.Context, twitchUserID string) error {
	_, err := m.dbPool.Query(ctx, "delete from "+
		"bot_users "+
		"where "+
		"twitch_user_id = $1;", twitchUserID)
	if err != nil {
		return err
	}

	_, err = m.dbPool.Query(ctx, "delete from "+
		"event_sub_subscriptions "+
		"where "+
		"twitch_user_id = $1;", twitchUserID)
	if err != nil {
		return err
	}

	_, err = m.dbPool.Query(ctx, "delete from "+
		"bot_commands "+
		"where "+
		"twitch_user_id = $1;", twitchUserID)
	return err
}
