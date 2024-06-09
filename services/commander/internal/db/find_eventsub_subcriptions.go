package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (m *mainDB) FindEventSubSubscriptionByID(ctx context.Context, eventSubID string) (*EventSubSubscription, bool, error) {
	rows, err := m.dbPool.Query(ctx, "select "+
		"subscription_id,topic,twitch_user_id "+
		"from "+
		"event_sub_subscriptions "+
		"where subscription_id = $1;", eventSubID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	defer rows.Close()

	sub := EventSubSubscription{}
	err = rows.Scan(&sub.SubscriptionID, &sub.Topic, &sub.TwitchUserID)
	if err != nil {
		return nil, false, err
	}

	return &sub, true, nil
}

func (m *mainDB) FindEventSubSubscriptionsForTwitchUserID(ctx context.Context, twitchUserID string) (*[]EventSubSubscription, error) {
	rows, err := m.dbPool.Query(ctx, "select "+
		"subscription_id,topic "+
		"from "+
		"event_sub_subscriptions "+
		"where twitch_user_id = $1;", twitchUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []EventSubSubscription

	for rows.Next() {
		sub := EventSubSubscription{TwitchUserID: twitchUserID}
		err = rows.Scan(&sub.SubscriptionID, &sub.Topic)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}

	return &subs, nil
}
