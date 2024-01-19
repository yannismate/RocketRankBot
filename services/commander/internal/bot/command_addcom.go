package bot

import (
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/rpc/commander"
	"context"
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

func (b *bot) executeCommandAddcom(ctx context.Context, req *commander.ExecutePossibleCommandReq) {
	var channelID string

	if req.TwitchChannelLogin == b.botChannelName {
		channelID = req.TwitchSenderUserID
	} else {
		channelID = req.TwitchChannelID
	}

	args := strings.Split(req.Command, " ")
	if len(args) != 4 {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageAddcomUsage, &req.TwitchMessageID)
		return
	}

	_, found, err := b.mainDB.FindUser(ctx, channelID)
	if err != nil {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInternalError, &req.TwitchMessageID)
		return
	}
	if !found {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageBotNotJoined, &req.TwitchMessageID)
		return
	}

	commandName := strings.ToLower(args[1])

	if _, ok := b.configCommands[commandName]; ok {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageCommandNameTaken, &req.TwitchMessageID)
		return
	}

	_, found, err = b.mainDB.FindCommand(ctx, channelID, commandName)
	if err != nil {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInternalError, &req.TwitchMessageID)
		return
	}
	if found {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageCommandNameTaken, &req.TwitchMessageID)
		return
	}

	platform := strings.ToLower(args[2])
	if _, ok := db.AllPlatforms[platform]; !ok {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInvalidPlatform, &req.TwitchMessageID)
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
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageInternalError, &req.TwitchMessageID)
		return
	}

	b.sendTwitchMessage(ctx, req.TwitchChannelLogin, messageCommandAdded, &req.TwitchMessageID)
}
