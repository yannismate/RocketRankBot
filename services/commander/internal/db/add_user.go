package db

import "context"

func (m *mainDB) AddUser(ctx context.Context, user *BotUser) error {
	res, err := m.dbPool.Query(ctx, "insert into "+
		"bot_users "+
		"(twitch_user_id, is_authenticated) "+
		"values "+
		"($1, $2);", user.TwitchUserID, user.IsAuthenticated)
	if err == nil {
		res.Close()
	}
	return err
}
