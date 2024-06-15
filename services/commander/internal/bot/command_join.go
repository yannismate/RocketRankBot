package bot

import (
	"context"
)

const (
	messageAlreadyJoined  = "The bot has already joined your channel. You can add a rank command using !addcom."
	messageReAuthRequired = "The bot is already configured for your channel, but is missing permissions to send messages in your channel. Please reauthenticate: "
	messageJoinAuth       = "To allow the bot to join your channel please authenticate here: "
)

func (b *bot) executeCommandJoin(ctx context.Context, req *IncomingPossibleCommand) {
	channelID := req.SenderID

	if req.ChannelID != b.botChannelID && !req.IsBroadcaster {
		b.sendTwitchMessage(ctx, req.ChannelID, messageBroadcasterOnly, &req.MessageID)
		return
	}
	dbUser, found, _ := b.mainDB.FindUser(ctx, channelID)
	if found {
		if !dbUser.IsAuthenticated {
			b.sendTwitchMessage(ctx, req.ChannelID, messageReAuthRequired+b.baseURL+"/auth", &req.MessageID)
			return
		}
		b.sendTwitchMessage(ctx, req.ChannelID, messageAlreadyJoined, &req.MessageID)
		return
	}

	b.sendTwitchMessage(ctx, req.ChannelID, messageJoinAuth+b.baseURL+"/auth", &req.MessageID)
}
