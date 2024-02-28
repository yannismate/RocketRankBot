package db

import "context"

func (m *mainDB) FindAllTwitchLogins(ctx context.Context) (*[]string, error) {
	rows, err := m.dbPool.Query(ctx, "select "+
		"twitch_login "+
		"from "+
		"bot_users;")
	if err != nil {
		return nil, err
	}

	channels := new([]string)

	err = rows.Scan(channels)
	if err != nil {
		return nil, err
	}

	return channels, nil
}
