package bot

import (
	"context"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	messageDelcomUsage    = "Unexpected Arguments. Usage: !addcom [command] [platform] [username]"
	messageCommandDeleted = "Command successfully deleted."
)

func (b *bot) executeCommandDelcom(ctx context.Context, req *IncomingPossibleCommand) {
	var channelID string

	if req.ChannelID == b.botChannelID {
		channelID = req.SenderID
	} else {
		channelID = req.ChannelID
	}

	args := strings.Split(req.Command, " ")
	if len(args) != 2 {
		b.sendTwitchMessage(ctx, req.ChannelID, messageDelcomUsage, &req.MessageID)
		return
	}

	commandName := strings.TrimPrefix(strings.ToLower(args[1]), b.commandPrefix)

	_, found, err := b.mainDB.FindCommand(ctx, channelID, commandName)
	if err != nil {
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for command")
		return
	}

	if !found {
		b.sendTwitchMessage(ctx, req.ChannelID, messageCommandDoesNotExist, &req.MessageID)
		return
	}

	err = b.mainDB.DeleteCommand(ctx, channelID, commandName)
	if err != nil {
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		log.Ctx(ctx).Error().Err(err).Msg("Could not delete command from db")
		return
	}

	err = b.cacheDB.InvalidateCachedCommand(ctx, channelID, commandName)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Could not invalidate cached command")
	}

	b.sendTwitchMessage(ctx, req.ChannelID, messageCommandDeleted, &req.MessageID)
}
