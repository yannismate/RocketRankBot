package connector

import (
	"RocketRankBot/services/twitchconnector/internal/auth"
	"RocketRankBot/services/twitchconnector/internal/config"
	"RocketRankBot/services/twitchconnector/internal/metrics"
	"RocketRankBot/services/twitchconnector/rpc/commander"
	"context"
	"errors"
	"github.com/gempir/go-twitch-irc/v4"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"net/http"
	"strings"
	"time"
)

const (
	minRetryInterval        = time.Second * 3
	maxRetryInterval        = time.Second * 60
	multiplierRetryInterval = 3
)

var (
	TwitchClientNotConnectedErr = errors.New("twitch client not connected")
)

type Connector interface {
	Start()
	IsConnected() bool
	JoinChannel(channel string) error
	LeaveChannel(channel string) error
	SendMessage(channel string, msg string) error
	SendResponseMessage(channel string, msg string, parentMsgID string) error
}

type connector struct {
	commanderClient commander.Commander
	twitchAuth      auth.TwitchAuth
	twitchLogin     string
	retryInterval   time.Duration
	commandPrefix   string
	twitchClient    *twitch.Client
	isConnected     bool
}

func NewConnector(cfg *config.TwitchconnectorConfig, commander commander.Commander) Connector {
	twitchAuth := auth.NewTwitchAuth(cfg)

	return &connector{
		commanderClient: commander,
		twitchAuth:      twitchAuth,
		twitchLogin:     cfg.Twitch.Login,
		commandPrefix:   cfg.CommandPrefix,
	}
}

func (c *connector) IsConnected() bool {
	return c.isConnected
}

func (c *connector) Start() {
	go func() {
		for {
			log.Info().Msg("Starting connector logic")
			err := c.tryStart()
			log.Error().Err(err).Int64("retry-in-ms", c.retryInterval.Milliseconds()).Msg("Connector failed")
			time.Sleep(c.retryInterval)
			c.retryInterval = max(c.retryInterval*multiplierRetryInterval, maxRetryInterval)
		}
	}()
}

func (c *connector) JoinChannel(channel string) error {
	if c.twitchClient == nil {
		return TwitchClientNotConnectedErr
	}
	c.twitchClient.Join(channel)
	metrics.GaugeJoinedChannels.Inc()
	return nil
}

func (c *connector) LeaveChannel(channel string) error {
	if c.twitchClient == nil {
		return TwitchClientNotConnectedErr
	}
	c.twitchClient.Depart(channel)
	metrics.GaugeJoinedChannels.Dec()
	return nil
}

func (c *connector) SendMessage(channel string, msg string) error {
	if c.twitchClient == nil {
		return TwitchClientNotConnectedErr
	}
	c.twitchClient.Say(channel, msg)
	metrics.CounterSentMessages.Inc()
	return nil
}

func (c *connector) SendResponseMessage(channel string, msg string, parentMsgID string) error {
	if c.twitchClient == nil {
		return TwitchClientNotConnectedErr
	}
	c.twitchClient.Reply(channel, parentMsgID, msg)
	metrics.CounterSentMessages.Inc()
	return nil
}

func (c *connector) tryStart() error {
	defer func() {
		c.isConnected = false
		metrics.GaugeJoinedChannels.Set(0)
	}()
	ctx := newBotContext()

	twitchToken, err := c.twitchAuth.GetAccessToken()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("could not fetch twitch auth token")
		return err
	}

	c.twitchClient = twitch.NewClient(c.twitchLogin, *twitchToken)
	c.twitchClient.SetJoinRateLimiter(twitch.CreateVerifiedRateLimiter())
	c.twitchClient.OnPrivateMessage(c.handleMessage)
	c.twitchClient.OnConnect(func() {
		c.isConnected = true
		metrics.GaugeJoinedChannels.Set(0)

		channelRes, err := c.commanderClient.GetAllChannels(ctx, &commander.GetAllChannelsReq{})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("could not fetch channels from commander")
			_ = c.twitchClient.Disconnect()
			return
		}

		currentOffset := 0
		for {
			toOffset := min(len(channelRes.TwitchChannelLogin)-1, currentOffset+1999)
			log.Ctx(ctx).Info().Int("from-offset", currentOffset).Int("to-offset", toOffset).Msg("Executing channel joins")
			c.twitchClient.Join(channelRes.TwitchChannelLogin[currentOffset:toOffset]...)
			metrics.GaugeJoinedChannels.Add(float64(toOffset - currentOffset))

			currentOffset += 2000
			if currentOffset >= len(channelRes.TwitchChannelLogin) {
				break
			}
			time.Sleep(time.Second * 10)
		}
		c.retryInterval = minRetryInterval
	})
	return c.twitchClient.Connect()
}

func (c *connector) handleMessage(msg twitch.PrivateMessage) {
	metrics.CounterReceivedMessages.Inc()

	cmd, isCmd := strings.CutPrefix(msg.Message, c.commandPrefix)
	if !isCmd {
		cmd, isCmd = strings.CutPrefix(strings.ToLower(msg.Message), "@"+c.twitchLogin+" "+c.commandPrefix)
		if !isCmd {
			return
		}
	}

	metrics.CounterReceivedPossibleCommands.Inc()

	ctx := newBotContext()
	isMod := false
	if modTag, ok := msg.Tags["mod"]; modTag == "1" && ok {
		isMod = true
	}

	isBroadcaster := msg.User.ID == msg.RoomID

	req := commander.ExecutePossibleCommandReq{
		TwitchChannelID:         msg.RoomID,
		TwitchChannelLogin:      strings.ToLower(msg.Channel),
		TwitchMessageID:         msg.ID,
		TwitchSenderUserID:      msg.User.ID,
		TwitchSenderDisplayName: msg.User.DisplayName,
		IsModerator:             isMod,
		IsBroadcaster:           isBroadcaster,
		Command:                 cmd,
	}
	_, err := c.commanderClient.ExecutePossibleCommand(ctx, &req)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error sending command execution to commander")
		return
	}
}

func newBotContext() context.Context {
	ctx := context.Background()

	traceId := uuid.New().String()
	spanId := uuid.New().String()
	ctx = context.WithValue(ctx, "trace-id", traceId)
	ctx = context.WithValue(ctx, "span-id", spanId)

	ctxLogger := log.With().Str("trace-id", traceId).Str("span-id", spanId).Logger()
	ctx = ctxLogger.WithContext(ctx)

	outgoingHeaders := make(http.Header)
	outgoingHeaders.Set("trace-id", traceId)
	outgoingHeaders.Set("span-id", spanId)
	ctx, err := twirp.WithHTTPRequestHeaders(ctx, outgoingHeaders)
	if err != nil {
		ctxLogger.Panic().Err(err)
	}

	return ctx
}
