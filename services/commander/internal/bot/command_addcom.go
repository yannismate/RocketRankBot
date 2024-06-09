package bot

import (
	"RocketRankBot/services/commander/internal/db"
	"context"
	"github.com/rs/zerolog/log"
	"strings"
)

const (
	messageAddcomUsage        = "Unexpected Arguments. Usage: !addcom [command] [platform] [username]"
	messageCommandNameTaken   = "This command name is already in use."
	messageCommandAdded       = "Command successfully added!"
	addcomDefaultFormat       = "Ranked 1v1: $(1.r) Div $(1.d) ($(1.m)) | Ranked 2v2: $(2.r) Div $(2.d) ($(2.m)) | Ranked 3v3: $(3.r) Div $(3.d) ($(3.m))"
	addcomDefaultCooldown     = 10
	addcomDefaultResponseType = db.TwitchResponseTypeMessage
)

func (b *bot) executeCommandAddcom(ctx context.Context, req *IncomingPossibleCommand) {
	var channelID string

	if req.ChannelID == b.botChannelID {
		channelID = req.SenderID
	} else {
		channelID = req.ChannelID
	}

	args := strings.Split(req.Command, " ")
	if len(args) < 4 {
		b.sendTwitchMessage(ctx, req.ChannelID, messageAddcomUsage, &req.MessageID)
		return
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

	commandName := strings.TrimPrefix(strings.ToLower(args[1]), b.commandPrefix)

	if _, ok := b.configCommands[commandName]; ok {
		b.sendTwitchMessage(ctx, req.ChannelID, messageCommandNameTaken, &req.MessageID)
		return
	}

	_, found, err = b.mainDB.FindCommand(ctx, channelID, commandName)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not query db for command")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}
	if found {
		b.sendTwitchMessage(ctx, req.ChannelID, messageCommandNameTaken, &req.MessageID)
		return
	}

	platform := strings.ToLower(args[2])
	if _, ok := db.AllPlatforms[platform]; !ok {
		b.sendTwitchMessage(ctx, req.ChannelID, messageInvalidPlatform, &req.MessageID)
		return
	}

	username := strings.Join(args[3:], " ")

	cmd := db.BotCommand{
		CommandName:            commandName,
		CommandCooldownSeconds: addcomDefaultCooldown,
		MessageFormat:          addcomDefaultFormat,
		TwitchUserID:           channelID,
		TwitchResponseType:     addcomDefaultResponseType,
		RLPlatform:             db.RLPlatform(platform),
		RLUsername:             username,
	}

	err = b.mainDB.AddCommand(ctx, &cmd)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not add command to db")
		b.sendTwitchMessage(ctx, req.ChannelID, getMessageInternalErrorWithCtx(ctx), &req.MessageID)
		return
	}

	b.sendTwitchMessage(ctx, req.ChannelID, messageCommandAdded, &req.MessageID)
}
