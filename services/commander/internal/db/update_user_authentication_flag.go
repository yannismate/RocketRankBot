package db

import "context"

func (m *mainDB) UpdateUserAuthenticationFlag(ctx context.Context, twitchUserID string, isAuthed bool) error {
	res, err := m.dbPool.Query(ctx, "update "+
		"bot_users "+
		"set "+
		"is_authenticated = $1 "+
		"where "+
		"twitch_user_id = $2;",
		isAuthed, twitchUserID)

	if err == nil {
		res.Close()
	}

	return err
}
