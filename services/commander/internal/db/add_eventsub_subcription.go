package db

import "context"

func (m *mainDB) AddEventSubSubscription(ctx context.Context, sub *EventSubSubscription) error {
	res, err := m.dbPool.Query(ctx, "insert into "+
		"event_sub_subscriptions "+
		"(subscription_id, twitch_user_id, topic) "+
		"values "+
		"($1, $2, $3);", sub.SubscriptionID, sub.TwitchUserID, sub.Topic)

	if err == nil {
		res.Close()
	}

	return err
}
