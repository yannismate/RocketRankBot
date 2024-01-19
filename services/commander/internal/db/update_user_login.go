package db

import "context"

func (m *mainDB) UpdateUserLogin(ctx context.Context, twitchUserID string, twitchLogin string) error {
	_, err := m.dbPool.Query(ctx, "update "+
		"bot_users "+
		"set "+
		"twitch_login = $1 "+
		"where "+
		"twitch_user_id = $2;", twitchLogin, twitchUserID)
	return err
}
