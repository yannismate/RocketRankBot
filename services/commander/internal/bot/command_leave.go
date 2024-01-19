package bot

import (
	"RocketRankBot/services/commander/rpc/commander"
	"context"
)

const messageBotLeft = "The bot has left your channel."

func (b *bot) executeCommandLeave(ctx context.Context, req *commander.ExecutePossibleCommandReq) {
	channelID := req.TwitchSenderUserID

	if req.TwitchChannelLogin != b.botChannelName && !req.IsBroadcaster {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageBroadcasterOnly, &req.TwitchMessageID)
		return
	}

	_, found, err := b.mainDB.FindUser(ctx, channelID)
	if !found {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageBotNotJoined, &req.TwitchMessageID)
		return
	}

	err = b.mainDB.DeleteUserData(ctx, channelID)
	if err != nil {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInternalError, &req.TwitchMessageID)
		return
	}

	b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageBotLeft, &req.TwitchMessageID)
	b.leaveTwitchChannel(ctx, channelID)
}
