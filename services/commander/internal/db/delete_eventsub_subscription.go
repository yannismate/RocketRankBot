package db

import "context"

func (m *mainDB) DeleteEventSubSubscription(ctx context.Context, subscriptionID string) error {
	_, err := m.dbPool.Query(ctx, "delete from "+
		"event_sub_subscriptions "+
		"where "+
		"subscription_id = $1;",
		subscriptionID)

	return err
}
