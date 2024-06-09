package bot

import (
	"context"
	"github.com/rs/zerolog/log"
)

const messageBotLeft = "The bot has left your channel."

func (b *bot) executeCommandLeave(ctx context.Context, req *IncomingPossibleCommand) {
	channelID := req.SenderID

	if req.ChannelID != b.botChannelID && !req.IsBroadcaster {
		b.sendTwitchMessage(ctx, req.ChannelLogin, messageBroadcasterOnly, &req.MessageID)
		return
	}

	_, found, err := b.mainDB.FindUser(ctx, channelID)
	if !found {
		b.sendTwitchMessage(ctx, req.ChannelLogin, messageBotNotJoined, &req.MessageID)
		return
	}

	subs, err := b.mainDB.FindEventSubSubscriptionsForTwitchUserID(ctx, channelID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not get event sub data from db")
		b.sendTwitchMessage(ctx, req.ChannelLogin, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}

	for _, sub := range *subs {
		err = b.twitchAPI.DeleteEventSubSubscription(ctx, sub.SubscriptionID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Could not delete user event sub")
		}
	}

	err = b.mainDB.DeleteUserData(ctx, channelID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not delete user data from db")
		b.sendTwitchMessage(ctx, req.ChannelLogin, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}

	b.sendTwitchMessage(ctx, req.ChannelLogin, messageBotLeft, &req.MessageID)
}
