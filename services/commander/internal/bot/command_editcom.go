package bot

import (
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/rpc/commander"
	"context"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
)

const (
	messageEditcomUsage          = "Unexpected Arguments. Usage: !editcom [command] [property] [values...]"
	messageEditcomAccountUsage   = "Unexpected Arguments. Usage: !editcom [command] account [platform] [username]"
	messageEditcomActionUsage    = "Unexpected Arguments. Usage: !editcom [command] action [reply action]"
	messageEditcomCooldownUsage  = "Unexpected Arguments. Usage: !editcom [command] cooldown [seconds]"
	messageCommandUpdated        = "Updated command successfully!"
	messageAddcomInvalidProperty = "Invalid property. Available properties: account, action, cooldown, format"
	messageInvalidReplyAction    = "Invalid reply action. Available actions: message, reply, mention"
	messageMinCooldown           = "The minimum cooldown for commands is 5 seconds."
	commandMinCooldown           = 5
)

func (b *bot) executeCommandEditcom(ctx context.Context, req *commander.ExecutePossibleCommandReq) {
	var channelID string

	if req.TwitchChannelLogin == b.botChannelName {
		channelID = req.TwitchSenderUserID
	} else {
		channelID = req.TwitchChannelID
	}

	args := strings.Split(req.Command, " ")

	if len(args) < 4 {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageEditcomUsage, &req.TwitchMessageID)
		return
	}

	commandName := strings.ToLower(args[1])
	property := strings.ToLower(args[2])

	_, found, err := b.mainDB.FindUser(ctx, channelID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for user")
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, getMessageInternalErrorWithCtx(ctx), &req.TwitchMessageID)
		return
	}
	if !found {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageBotNotJoined, &req.TwitchMessageID)
		return
	}

	dbCmd, found, err := b.mainDB.FindCommand(ctx, channelID, commandName)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for command")
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, getMessageInternalErrorWithCtx(ctx), &req.TwitchMessageID)
		return
	}
	if !found {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageCommandDoesNotExist, &req.TwitchMessageID)
		return
	}

	switch property {
	case "account":
		if len(args) < 5 {
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageEditcomAccountUsage, &req.TwitchMessageID)
			return
		}

		platform := strings.ToLower(args[3])
		if _, ok := db.AllPlatforms[platform]; !ok {
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInvalidPlatform, &req.TwitchMessageID)
			return
		}
		newUserName := strings.Join(args[4:], " ")
		dbCmd.RLPlatform = db.RLPlatform(platform)
		dbCmd.RLUsername = newUserName

	case "action":
		if len(args) != 4 {
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageEditcomActionUsage, &req.TwitchMessageID)
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
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInvalidReplyAction, &req.TwitchMessageID)
			return
		}

	case "cooldown":
		newCooldown, err := strconv.Atoi(args[3])
		if len(args) != 4 || err != nil {
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageEditcomCooldownUsage, &req.TwitchMessageID)
			return
		}
		if newCooldown < commandMinCooldown {
			b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageMinCooldown, &req.TwitchMessageID)
			return
		}
		dbCmd.CommandCooldownSeconds = newCooldown

	case "format":
		formatStr := strings.Join(args[3:], " ")
		dbCmd.MessageFormat = formatStr

	default:
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageAddcomInvalidProperty, &req.TwitchMessageID)
		return
	}

	err = b.mainDB.UpdateCommand(ctx, dbCmd)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not update command in db")
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, getMessageInternalErrorWithCtx(ctx), &req.TwitchMessageID)
		return
	}
	err = b.cacheDB.InvalidateCachedCommand(ctx, dbCmd.TwitchUserID, dbCmd.CommandName)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Could not invalidate command in cache")
	}

	b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageCommandUpdated, &req.TwitchMessageID)
}
