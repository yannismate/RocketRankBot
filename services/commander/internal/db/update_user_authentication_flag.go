package db

import "context"

func (m *mainDB) UpdateUserAuthenticationFlag(ctx context.Context, twitchUserID string, isAuthed bool) error {
	_, err := m.dbPool.Query(ctx, "update "+
		"bot_users "+
		"set "+
		"is_authenticated = $1 "+
		"where "+
		"twitch_user_id = $2;",
		isAuthed, twitchUserID)

	return err
}
