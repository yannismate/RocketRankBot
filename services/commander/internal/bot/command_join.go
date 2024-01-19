package bot

import (
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/rpc/commander"
	"context"
	"strings"
)

const (
	messageAlreadyJoined = "The bot has already joined your channel. You can add a rank command using !addcmd."
	messageBotJoined     = "The bot has joined your channel. Add commands using !addcmd."
)

func (b *bot) executeCommandJoin(ctx context.Context, req *commander.ExecutePossibleCommandReq) {
	channelID := req.TwitchSenderUserID
	channelLogin := strings.ToLower(req.TwitchSenderDisplayName)

	if req.TwitchChannelLogin != b.botChannelName && !req.IsBroadcaster {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageBroadcasterOnly, &req.TwitchMessageID)
		return
	}
	oldUser, found, err := b.mainDB.FindUser(ctx, channelID)
	if found {
		if channelLogin == oldUser.TwitchLogin {
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageAlreadyJoined, &req.TwitchMessageID)
			return
		}
		err = b.mainDB.UpdateUserLogin(ctx, channelID, channelLogin)
		if err != nil {
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInternalError, &req.TwitchMessageID)
			return
		}
		b.leaveTwitchChannel(ctx, oldUser.TwitchLogin)
		b.joinTwitchChannel(ctx, channelLogin)
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageChannelNameUpdate, &req.TwitchMessageID)
		return
	}

	bu := db.BotUser{
		TwitchUserID: channelID,
		TwitchLogin:  channelLogin,
	}

	err = b.mainDB.AddUser(ctx, &bu)
	if err != nil {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInternalError, &req.TwitchMessageID)
		return
	}
	b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageBotJoined, &req.TwitchMessageID)
}
