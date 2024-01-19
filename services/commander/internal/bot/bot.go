package bot

import (
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/formatter"
	"RocketRankBot/services/commander/rpc/commander"
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"RocketRankBot/services/commander/rpc/twitchconnector"
	"bytes"
	"context"
	"errors"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"strings"
	"time"
)

type Bot interface {
	ExecutePossibleCommand(ctx context.Context, req *commander.ExecutePossibleCommandReq)
}

type bot struct {
	mainDB           db.MainDB
	cacheDB          db.CacheDB
	twitchConnector  twitchconnector.TwitchConnector
	trackerGgScraper trackerggscraper.TrackerGgScraper
	commandTimeout   time.Duration
	cacheTTLCommand  time.Duration
	cacheTTLRank     time.Duration
	botChannelName   string
	configCommands   map[string]func(ctx context.Context, req *commander.ExecutePossibleCommandReq)
}

func NewBot(mainDB db.MainDB, cacheDB db.CacheDB, cfg *config.CommanderConfig, tc twitchconnector.TwitchConnector, tgs trackerggscraper.TrackerGgScraper) Bot {
	b := bot{
		mainDB:           mainDB,
		cacheDB:          cacheDB,
		twitchConnector:  tc,
		trackerGgScraper: tgs,
		commandTimeout:   time.Second * time.Duration(cfg.CommandTimeoutSeconds),
		cacheTTLCommand:  time.Second * time.Duration(cfg.TTL.Commands),
		cacheTTLRank:     time.Second * time.Duration(cfg.TTL.Ranks),
		botChannelName:   cfg.BotChannelName,
	}
	b.configCommands = map[string]func(ctx context.Context, req *commander.ExecutePossibleCommandReq){
		"join":   b.executeCommandJoin,
		"leave":  b.executeCommandLeave,
		"addcom": b.executeCommandAddcom,
		// TODO: delete and edit commands, twitchconnector service
	}

	return &b
}

var (
	platformDBToProtoMapping = map[db.RLPlatform]trackerggscraper.PlayerPlatform{
		db.RLPlatformEpic:  trackerggscraper.PlayerPlatform_EPIC,
		db.RLPlatformSteam: trackerggscraper.PlayerPlatform_STEAM,
		db.RLPlatformPS:    trackerggscraper.PlayerPlatform_PSN,
		db.RLPlatformXbox:  trackerggscraper.PlayerPlatform_XBL,
	}
)

func (b *bot) ExecutePossibleCommand(ctx context.Context, req *commander.ExecutePossibleCommandReq) {
	ctx, cancel := context.WithTimeout(ctx, b.commandTimeout)
	defer cancel()

	commandParts := strings.Split(req.Command, " ")
	baseCommand := strings.ToLower(commandParts[0])

	if cmdFunc, ok := b.configCommands[strings.ToLower(baseCommand)]; ok {
		// Built-in commands can only be executed by mods/broadcasters or in the bots chat
		if strings.ToLower(req.TwitchChannelLogin) != strings.ToLower(b.botChannelName) && !req.IsModerator && !req.IsBroadcaster {
			return
		}
		cmdFunc(ctx, req)
		return
	}

	cachedCommand, foundCache, err := b.cacheDB.FindCachedCommand(ctx, req.TwitchChannelID, baseCommand)
	if err != nil {
		zap.L().Error("Error looking up cached command", zap.Error(err))
	}

	var replyMessage string
	var replyType db.TwitchResponseType
	var updatedCachedCmd db.CachedCommand

	if foundCache {
		if time.Now().Before(cachedCommand.NextExecutionAllowedTime) {
			return
		}
		replyMessage = b.getRankMessage(ctx, cachedCommand.RLPlatform, cachedCommand.RLUsername, cachedCommand.MessageFormat)
		replyType = cachedCommand.TwitchResponseType
		updatedCachedCmd = *cachedCommand
		updatedCachedCmd.NextExecutionAllowedTime = time.Now().Add(time.Second * time.Duration(cachedCommand.CommandCooldownSeconds))
	} else {
		command, foundMain, err := b.mainDB.FindCommand(ctx, req.TwitchChannelID, baseCommand)
		if err != nil {
			replyMessage = messageInternalError
			zap.L().Error("Error looking up command in DB", zap.Error(err))
			return
		}
		if !foundMain {
			return
		}

		replyType = command.TwitchResponseType
		replyMessage = b.getRankMessage(ctx, command.RLPlatform, command.RLUsername, command.MessageFormat)
		updatedCachedCmd = db.CachedCommand{
			CommandCooldownSeconds:   command.CommandCooldownSeconds,
			NextExecutionAllowedTime: time.Now().Add(time.Second * time.Duration(command.CommandCooldownSeconds)),
			MessageFormat:            command.MessageFormat,
			TwitchResponseType:       command.TwitchResponseType,
			RLPlatform:               command.RLPlatform,
			RLUsername:               command.RLUsername,
		}
	}

	err = b.cacheDB.SetCachedCommand(ctx, req.TwitchChannelID, baseCommand, &updatedCachedCmd, b.cacheTTLCommand)
	if err != nil {
		zap.L().Error("Error updating command cache", zap.Error(err))
	}

	if replyType == db.TwitchResponseTypeMention {
		if len(commandParts) > 1 {
			if strings.HasPrefix("@", commandParts[1]) {
				replyMessage = commandParts[1] + " " + replyMessage
			} else {
				replyMessage = "@" + commandParts[1] + " " + replyMessage
			}
		}
	}

	if replyType == db.TwitchResponseTypeReply {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, replyMessage, &req.TwitchMessageID)
	} else {
		b.sendTwitchMessage(ctx, req.TwitchChannelLogin, replyMessage, nil)
	}
}

func (b *bot) getRankMessage(ctx context.Context, platform db.RLPlatform, identifier string, format string) string {

	rankRes, wasCached, err := b.cacheDB.FindCachedRank(ctx, platform, identifier)
	if err != nil {
		zap.L().Error("Error looking up cached rank", zap.Error(err))
	}

	if !wasCached {
		rankReq := trackerggscraper.PlayerCurrentRanksReq{
			Platform:   platformDBToProtoMapping[platform],
			Identifier: identifier,
		}
		rankRes, err = b.trackerGgScraper.PlayerCurrentRanks(ctx, &rankReq)
		if err != nil {
			var twirpErr twirp.Error
			if errors.As(err, &twirpErr) {
				if twirpErr.Code() == twirp.ResourceExhausted {
					zap.L().Info("Rank service is rate limited", zap.Error(err))
					return messageRateLimited
				} else if twirpErr.Code() == twirp.NotFound {
					notFoundStruct := struct {
						PlayerName     string
						PlayerPlatform string
					}{
						PlayerName:     identifier,
						PlayerPlatform: string(platform),
					}
					var notFoundMessageBuf bytes.Buffer
					err = templateMessageNotFound.Execute(&notFoundMessageBuf, notFoundStruct)
					if err != nil {
						zap.L().Error("Error executing not found template", zap.Error(err))
						return messageInternalError
					}
					return notFoundMessageBuf.String()
				}
			}
			zap.L().Error("Error getting ranks from scraping service", zap.Error(err))
			return messageInternalError
		}

		err = b.cacheDB.SetCachedRank(ctx, platform, identifier, rankRes, b.cacheTTLRank)
		if err != nil {
			zap.L().Error("Error updating rank cache", zap.Error(err))
		}
	}

	return formatter.FormatRankString(rankRes, format)
}

func (b *bot) sendTwitchMessage(ctx context.Context, channelName string, message string, asReplyTo *string) {
	if asReplyTo == nil {
		sendMsgReq := twitchconnector.SendMessageReq{
			TwitchLogin: channelName,
			Message:     message,
		}
		_, err := b.twitchConnector.SendMessage(ctx, &sendMsgReq)
		if err != nil {
			zap.L().Error("Error sending message", zap.Error(err))
			return
		}
	} else {
		sendMsgReq := twitchconnector.SendResponseMessageReq{
			TwitchLogin:        channelName,
			Message:            message,
			RespondToMessageID: *asReplyTo,
		}
		_, err := b.twitchConnector.SendResponseMessage(ctx, &sendMsgReq)
		if err != nil {
			zap.L().Error("Error sending reply message", zap.Error(err))
			return
		}
	}
}

func (b *bot) joinTwitchChannel(ctx context.Context, channelLogin string) {
	joinChannelReq := twitchconnector.JoinChannelReq{
		TwitchLogin: channelLogin,
	}
	_, err := b.twitchConnector.JoinChannel(ctx, &joinChannelReq)
	if err != nil {
		zap.L().Error("Error sending join channel message", zap.Error(err))
		return
	}
}

func (b *bot) leaveTwitchChannel(ctx context.Context, channelLogin string) {
	leaveChannelReq := twitchconnector.LeaveChannelReq{
		TwitchLogin: channelLogin,
	}
	_, err := b.twitchConnector.LeaveChannel(ctx, &leaveChannelReq)
	if err != nil {
		zap.L().Error("Error sending leave channel message", zap.Error(err))
		return
	}
}
