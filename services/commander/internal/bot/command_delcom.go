package bot

import (
	"RocketRankBot/services/commander/rpc/commander"
	"context"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	messageDelcomUsage         = "Unexpected Arguments. Usage: !addcom [command] [platform] [username]"
	messageCommandDoesNotExist = "Command could not be found."
	messageCommandDeleted      = "Command successfully deleted."
)

func (b *bot) executeCommandDelcom(ctx context.Context, req *commander.ExecutePossibleCommandReq) {
	var channelID string

	if req.TwitchChannelLogin == b.botChannelName {
		channelID = req.TwitchSenderUserID
	} else {
		channelID = req.TwitchChannelID
	}

	args := strings.Split(req.Command, " ")
	if len(args) != 2 {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageDelcomUsage, &req.TwitchMessageID)
		return
	}

	commandName := strings.ToLower(args[1])

	_, found, err := b.mainDB.FindCommand(ctx, channelID, commandName)
	if err != nil {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, getMessageInternalErrorWithCtx(ctx), &req.TwitchMessageID)
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for command")
		return
	}

	if !found {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageCommandDoesNotExist, &req.TwitchMessageID)
		return
	}

	err = b.mainDB.DeleteCommand(ctx, channelID, commandName)
	if err != nil {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, getMessageInternalErrorWithCtx(ctx), &req.TwitchMessageID)
		log.Ctx(ctx).Error().Err(err).Msg("Could not delete command from db")
		return
	}

	err = b.cacheDB.InvalidateCachedCommand(ctx, channelID, commandName)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Could not invalidate cached command")
	}

	b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageCommandDeleted, &req.TwitchMessageID)
}
