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
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"strings"
	"time"
)

type Bot interface {
	ExecutePossibleCommand(ctx context.Context, req *commander.ExecutePossibleCommandReq)
	GetAllChannels(ctx context.Context) (*[]string, error)
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
		"delcom": b.executeCommandDelcom,
		// TODO: edit commands, twitchconnector service
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
		log.Ctx(ctx).Info().Str("channel-id", req.TwitchChannelID).Str("channel-login", req.TwitchChannelLogin).Str("sender-id", req.TwitchSenderUserID).Str("sender-login", strings.ToLower(req.TwitchSenderDisplayName)).Str("command", req.Command).Msg("Executing builtin command")
		cmdFunc(ctx, req)
		return
	}

	cachedCommand, foundCache, err := b.cacheDB.FindCachedCommand(ctx, req.TwitchChannelID, baseCommand)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error looking up cached command")
	}

	var replyMessage string
	var replyType db.TwitchResponseType
	var updatedCachedCmd db.CachedCommand

	if foundCache {
		if time.Now().Before(cachedCommand.NextExecutionAllowedTime) {
			return
		}
		log.Ctx(ctx).Info().Str("channel-id", req.TwitchChannelID).Str("channel-login", req.TwitchChannelLogin).Str("sender-id", req.TwitchSenderUserID).Str("sender-login", strings.ToLower(req.TwitchSenderDisplayName)).Str("command", req.Command).Msg("Executing cached rank command")
		replyMessage = b.getRankMessage(ctx, cachedCommand.RLPlatform, cachedCommand.RLUsername, cachedCommand.MessageFormat)
		replyType = cachedCommand.TwitchResponseType
		updatedCachedCmd = *cachedCommand
		updatedCachedCmd.NextExecutionAllowedTime = time.Now().Add(time.Second * time.Duration(cachedCommand.CommandCooldownSeconds))
	} else {
		command, foundMain, err := b.mainDB.FindCommand(ctx, req.TwitchChannelID, baseCommand)
		if err != nil {
			replyMessage = messageInternalError
			log.Ctx(ctx).Error().Err(err).Msg("Error looking up command in DB")
			return
		}
		if !foundMain {
			return
		}

		log.Ctx(ctx).Info().Str("channel-id", req.TwitchChannelID).Str("channel-login", req.TwitchChannelLogin).Str("sender-id", req.TwitchSenderUserID).Str("sender-login", strings.ToLower(req.TwitchSenderDisplayName)).Str("command", req.Command).Msg("Executing rank command")

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
		log.Ctx(ctx).Error().Err(err).Msg("Error updating command cache")
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

func (b *bot) GetAllChannels(ctx context.Context) (*[]string, error) {
	return b.mainDB.FindAllTwitchLogins(ctx)
}

func (b *bot) getRankMessage(ctx context.Context, platform db.RLPlatform, identifier string, format string) string {

	rankRes, wasCached, err := b.cacheDB.FindCachedRank(ctx, platform, identifier)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error looking up cached rank")
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
					log.Ctx(ctx).Info().Err(err).Msg("Rank service is rate limited")
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
						log.Ctx(ctx).Error().Err(err).Msg("Error executing not found template")
						return messageInternalError
					}
					return notFoundMessageBuf.String()
				}
			}
			log.Ctx(ctx).Error().Err(err).Msg("Error getting ranks from scraping service")
			return messageInternalError
		}

		err = b.cacheDB.SetCachedRank(ctx, platform, identifier, rankRes, b.cacheTTLRank)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Error updating rank cache")
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
			log.Ctx(ctx).Error().Err(err).Msg("Error sending message")
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
			log.Ctx(ctx).Error().Err(err).Msg("Error sending reply message")
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
		log.Ctx(ctx).Error().Err(err).Msg("Error sending join channel message")
		return
	}
}

func (b *bot) leaveTwitchChannel(ctx context.Context, channelLogin string) {
	leaveChannelReq := twitchconnector.LeaveChannelReq{
		TwitchLogin: channelLogin,
	}
	_, err := b.twitchConnector.LeaveChannel(ctx, &leaveChannelReq)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error sending leave channel message")
		return
	}
}
