package bot

import (
	"context"
	"github.com/rs/zerolog/log"
)

const (
	messageNoCommandsConfigured = "No commands are configured for this channel."
)

func (b *bot) executeCommandListcom(ctx context.Context, req *IncomingPossibleCommand) {
	var channelID string

	if req.ChannelID == b.botChannelID {
		channelID = req.SenderID
	} else {
		channelID = req.ChannelID
	}

	_, found, err := b.mainDB.FindUser(ctx, channelID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for user")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}
	if !found {
		b.sendTwitchMessage(ctx, req.ChannelID, messageBotNotJoined, &req.MessageID)
		return
	}

	commands, err := b.mainDB.FindUserCommands(ctx, channelID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for commands")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}
	if len(*commands) == 0 {
		b.sendTwitchMessage(ctx, req.ChannelID, messageNoCommandsConfigured, &req.MessageID)
		return
	}

	resMsg := "Commands in this channel: "

	firstCmdInList := true
	for _, cmd := range *commands {
		if len(resMsg)+len(cmd.CommandName)+3 > 500 {
			b.sendTwitchMessage(ctx, req.ChannelID, resMsg, &req.MessageID)
			resMsg = ""
			firstCmdInList = false
		}
		if firstCmdInList {
			resMsg += "!" + cmd.CommandName
		} else {
			resMsg += ", !" + cmd.CommandName
		}
	}
	b.sendTwitchMessage(ctx, req.ChannelID, resMsg, &req.MessageID)
}
