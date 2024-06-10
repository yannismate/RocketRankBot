package bot

import (
	"RocketRankBot/services/commander/internal/db"
	"context"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
)

const (
	messageEditcomUsage          = "Unexpected Arguments. Usage: !editcom [command] [account/action/cooldown/format] [values...]"
	messageEditcomAccountUsage   = "Unexpected Arguments. Usage: !editcom [command] account [platform] [username]"
	messageEditcomActionUsage    = "Unexpected Arguments. Usage: !editcom [command] action [reply action]"
	messageEditcomCooldownUsage  = "Unexpected Arguments. Usage: !editcom [command] cooldown [seconds]"
	messageCommandUpdated        = "Updated command successfully!"
	messageAddcomInvalidProperty = "Invalid property. Available properties: account, action, cooldown, format"
	messageInvalidReplyAction    = "Invalid reply action. Available actions: message, reply, mention"
	messageMinCooldown           = "The minimum cooldown for commands is 5 seconds."
	commandMinCooldown           = 5
)

func (b *bot) executeCommandEditcom(ctx context.Context, req *IncomingPossibleCommand) {
	var channelID string

	if req.ChannelID == b.botChannelID {
		channelID = req.SenderID
	} else {
		channelID = req.ChannelID
	}

	args := strings.Split(req.Command, " ")

	if len(args) < 4 {
		b.sendTwitchMessage(ctx, req.ChannelID, messageEditcomUsage, &req.MessageID)
		return
	}

	commandName := strings.TrimPrefix(strings.ToLower(args[1]), b.commandPrefix)
	property := strings.ToLower(args[2])

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

	dbCmd, found, err := b.mainDB.FindCommand(ctx, channelID, commandName)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for command")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}
	if !found {
		b.sendTwitchMessage(ctx, req.ChannelID, messageCommandDoesNotExist, &req.MessageID)
		return
	}

	switch property {
	case "account":
		if len(args) < 5 {
			b.sendTwitchMessage(ctx, req.ChannelID, messageEditcomAccountUsage, &req.MessageID)
			return
		}

		platform := strings.ToLower(args[3])
		if _, ok := db.AllPlatforms[platform]; !ok {
			b.sendTwitchMessage(ctx, req.ChannelID, messageInvalidPlatform, &req.MessageID)
			return
		}
		newUserName := strings.Join(args[4:], " ")
		dbCmd.RLPlatform = db.RLPlatform(platform)
		dbCmd.RLUsername = newUserName

	case "action":
		if len(args) != 4 {
			b.sendTwitchMessage(ctx, req.ChannelID, messageEditcomActionUsage, &req.MessageID)
			return
		}
		action := strings.ToLower(args[3])
		switch action {
		case "message":
			dbCmd.TwitchResponseType = db.TwitchResponseTypeMessage
		case "reply":
			dbCmd.TwitchResponseType = db.TwitchResponseTypeReply
		case "mention":
			dbCmd.TwitchResponseType = db.TwitchResponseTypeMention
		default:
			b.sendTwitchMessage(ctx, req.ChannelID, messageInvalidReplyAction, &req.MessageID)
			return
		}

	case "cooldown":
		newCooldown, err := strconv.Atoi(args[3])
		if len(args) != 4 || err != nil {
			b.sendTwitchMessage(ctx, req.ChannelID, messageEditcomCooldownUsage, &req.MessageID)
			return
		}
		if newCooldown < commandMinCooldown {
			b.sendTwitchMessage(ctx, req.ChannelID, messageMinCooldown, &req.MessageID)
			return
		}
		dbCmd.CommandCooldownSeconds = newCooldown

	case "format":
		formatStr := strings.Join(args[3:], " ")
		dbCmd.MessageFormat = formatStr

	default:
		b.sendTwitchMessage(ctx, req.ChannelID, messageAddcomInvalidProperty, &req.MessageID)
		return
	}

	err = b.mainDB.UpdateCommand(ctx, dbCmd)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not update command in db")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}
	err = b.cacheDB.InvalidateCachedCommand(ctx, dbCmd.TwitchUserID, dbCmd.CommandName)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Could not invalidate command in cache")
	}

	b.sendTwitchMessage(ctx, req.ChannelID, messageCommandUpdated, &req.MessageID)
}
