package bot

import (
	"context"
	"fmt"
)

const (
	messageAlreadyJoined  = "The bot is already in your channel. You can add a rank command using !addcom."
	messageReAuthRequired = "The bot is already configured for your channel, but is missing permissions to send messages. Please reauthenticate: %s"
	messageJoinAuth       = "To allow the bot to join your channel please authenticate here: %s"
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
			b.sendTwitchMessage(ctx, req.ChannelID, fmt.Sprintf(messageReAuthRequired, b.baseURL+"/auth"), &req.MessageID)
			return
		}
		b.sendTwitchMessage(ctx, req.ChannelID, messageAlreadyJoined, &req.MessageID)
		return
	}

	b.sendTwitchMessage(ctx, req.ChannelID, fmt.Sprintf(messageJoinAuth, b.baseURL+"/auth"), &req.MessageID)
}
