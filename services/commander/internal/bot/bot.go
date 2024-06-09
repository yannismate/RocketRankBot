package bot

import (
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/formatter"
	"RocketRankBot/services/commander/internal/metrics"
	"RocketRankBot/services/commander/internal/twitch"
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"bytes"
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"strings"
	"time"
)

type Bot interface {
	ExecutePossibleCommand(ctx context.Context, req *IncomingPossibleCommand)
}

type bot struct {
	mainDB           db.MainDB
	cacheDB          db.CacheDB
	twitchAPI        twitch.API
	baseURL          string
	trackerGgScraper trackerggscraper.TrackerGgScraper
	commandTimeout   time.Duration
	cacheTTLCommand  time.Duration
	cacheTTLRank     time.Duration
	botChannelID     string
	configCommands   map[string]func(ctx context.Context, req *IncomingPossibleCommand)
}

type IncomingPossibleCommand struct {
	Command        string
	IsModerator    bool
	IsBroadcaster  bool
	ChannelID      string
	ChannelLogin   string
	SenderID       string
	SenderLogin    string
	MessageID      string
	UsedPingPrefix bool
}

func NewBot(mainDB db.MainDB, cacheDB db.CacheDB, cfg *config.CommanderConfig, ta twitch.API, tgs trackerggscraper.TrackerGgScraper) Bot {
	b := bot{
		mainDB:           mainDB,
		cacheDB:          cacheDB,
		twitchAPI:        ta,
		baseURL:          cfg.BaseURL,
		trackerGgScraper: tgs,
		commandTimeout:   time.Second * time.Duration(cfg.CommandTimeoutSeconds),
		cacheTTLCommand:  time.Second * time.Duration(cfg.TTL.Commands),
		cacheTTLRank:     time.Second * time.Duration(cfg.TTL.Ranks),
		botChannelID:     cfg.Twitch.BotUserID,
	}
	b.configCommands = map[string]func(ctx context.Context, req *IncomingPossibleCommand){
		"join":    b.executeCommandJoin,
		"leave":   b.executeCommandLeave,
		"addcom":  b.executeCommandAddcom,
		"delcom":  b.executeCommandDelcom,
		"editcom": b.executeCommandEditcom,
		// TODO: listcom
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

func (b *bot) ExecutePossibleCommand(ctx context.Context, req *IncomingPossibleCommand) {
	executionStartedAt := time.Now()
	ctx, cancel := context.WithTimeout(ctx, b.commandTimeout)
	defer cancel()

	commandParts := strings.Split(req.Command, " ")
	baseCommand := strings.ToLower(commandParts[0])

	if cmdFunc, ok := b.configCommands[strings.ToLower(baseCommand)]; ok {
		// Built-in commands can only be executed by mods/broadcasters using the ping prefix or in the bots chat
		if req.ChannelID != b.botChannelID && !req.IsModerator && !req.IsBroadcaster {
			return
		}
		if req.ChannelID != b.botChannelID && !req.UsedPingPrefix {
			return
		}

		defer metrics.HistogramCommandResponseTime.With(prometheus.Labels{"type": "builtin"}).Observe(float64(time.Now().UnixMilli() - executionStartedAt.UnixMilli()))
		metrics.CounterExecutedCommandsBuiltin.Inc()

		log.Ctx(ctx).Info().Str("channel-id", req.ChannelID).Str("channel-login", req.ChannelLogin).Str("sender-id", req.SenderID).Str("sender-login", strings.ToLower(req.SenderLogin)).Str("command", req.Command).Msg("Executing builtin command")
		cmdFunc(ctx, req)
		return
	}

	cachedCommand, foundCache, err := b.cacheDB.FindCachedCommand(ctx, req.ChannelID, baseCommand)
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
		defer metrics.HistogramCommandResponseTime.With(prometheus.Labels{"type": "rank"}).Observe(float64(time.Now().UnixMilli() - executionStartedAt.UnixMilli()))
		metrics.CounterExecutedCommandsRank.Inc()
		metrics.CounterCachedCommandsRank.Inc()

		log.Ctx(ctx).Info().Str("channel-id", req.ChannelID).Str("channel-login", req.ChannelLogin).Str("sender-id", req.SenderID).Str("sender-login", strings.ToLower(req.SenderLogin)).Str("command", req.Command).Msg("Executing cached rank command")
		replyMessage = b.getRankMessage(ctx, cachedCommand.RLPlatform, cachedCommand.RLUsername, cachedCommand.MessageFormat)
		replyType = cachedCommand.TwitchResponseType
		updatedCachedCmd = *cachedCommand
		updatedCachedCmd.NextExecutionAllowedTime = time.Now().Add(time.Second * time.Duration(cachedCommand.CommandCooldownSeconds))
	} else {
		command, foundMain, err := b.mainDB.FindCommand(ctx, req.ChannelID, baseCommand)
		if err != nil {
			replyMessage = getMessageInternalErrorWithCtx(ctx)
			log.Ctx(ctx).Error().Err(err).Msg("Error looking up command in DB")
			return
		}
		if !foundMain {
			return
		}

		defer metrics.HistogramCommandResponseTime.With(prometheus.Labels{"type": "rank"}).Observe(float64(time.Now().UnixMilli() - executionStartedAt.UnixMilli()))
		metrics.CounterExecutedCommandsRank.Inc()
		log.Ctx(ctx).Info().Str("channel-id", req.ChannelID).Str("channel-login", req.ChannelLogin).Str("sender-id", req.SenderID).Str("sender-login", strings.ToLower(req.SenderLogin)).Str("command", req.Command).Msg("Executing rank command")

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

	err = b.cacheDB.SetCachedCommand(ctx, req.ChannelID, baseCommand, &updatedCachedCmd, b.cacheTTLCommand)
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
		b.sendTwitchMessage(ctx, req.ChannelID, replyMessage, &req.MessageID)
	} else {
		b.sendTwitchMessage(ctx, req.ChannelID, replyMessage, nil)
	}
}

func (b *bot) getRankMessage(ctx context.Context, platform db.RLPlatform, identifier string, format string) string {

	rankRes, wasCached, err := b.cacheDB.FindCachedRank(ctx, platform, identifier)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error looking up cached rank")
	}
	if wasCached {
		metrics.CounterCachedRequestsRank.Inc()
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
						return getMessageInternalErrorWithCtx(ctx)
					}
					return notFoundMessageBuf.String()
				}
			}
			log.Ctx(ctx).Error().Err(err).Msg("Error getting ranks from scraping service")
			return getMessageInternalErrorWithCtx(ctx)
		}

		err = b.cacheDB.SetCachedRank(ctx, platform, identifier, rankRes, b.cacheTTLRank)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Error updating rank cache")
		}
	}

	return formatter.FormatRankString(rankRes, format)
}

func (b *bot) sendTwitchMessage(ctx context.Context, channelID string, message string, asReplyTo *string) {
	err := b.twitchAPI.SendChatMessage(ctx, channelID, message, asReplyTo)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error sending twitch message")
		return
	}
}
