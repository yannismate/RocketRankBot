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
	defer rows.Close()

	var channels []string

	for rows.Next() {
		var ch string
		err = rows.Scan(&ch)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}

	return &channels, nil
}
