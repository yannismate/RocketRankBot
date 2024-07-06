package bot

import (
	"context"
	"github.com/rs/zerolog/log"
)

const messageBotLeft = "The bot has left your channel."

func (b *bot) executeCommandLeave(ctx context.Context, req *IncomingPossibleCommand) {
	channelID := req.SenderID

	if req.ChannelID != b.botChannelID && !req.IsBroadcaster && !req.IsAdmin {
		b.sendTwitchMessage(ctx, req.ChannelID, messageBroadcasterOnly, &req.MessageID)
		return
	}

	_, found, err := b.mainDB.FindUser(ctx, channelID)
	if !found {
		b.sendTwitchMessage(ctx, req.ChannelID, messageBotNotJoined, &req.MessageID)
		return
	}

	subs, err := b.mainDB.FindEventSubSubscriptionsForTwitchUserID(ctx, channelID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not get event sub data from db")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}

	err = b.mainDB.DeleteUserData(ctx, channelID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not delete user data from db")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}

	b.sendTwitchMessage(ctx, req.ChannelID, messageBotLeft, &req.MessageID)

	// Delete EventSub subscriptions last to allow response to go through
	for _, sub := range *subs {
		err = b.twitchAPI.DeleteEventSubSubscription(ctx, sub.SubscriptionID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Could not delete user event sub")
		}
	}

	log.Ctx(ctx).Info().Str("user_id", channelID).Msg("Finished deleting EventSub Subscriptions for user deletion.")
}
