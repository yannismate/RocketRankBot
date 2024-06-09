package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (m *mainDB) FindUser(ctx context.Context, twitchUserID string) (*BotUser, bool, error) {
	bu := BotUser{}

	err := m.dbPool.QueryRow(ctx, "select "+
		"twitch_user_id, is_authenticated "+
		"from bot_users "+
		"where "+
		"twitch_user_id = $1;",
		twitchUserID).Scan(&bu.TwitchUserID, &bu.IsAuthenticated)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &bu, true, nil
}
