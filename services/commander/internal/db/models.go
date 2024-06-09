package db

import "time"

type RLPlatform string

const (
	RLPlatformEpic  RLPlatform = "epic"
	RLPlatformSteam RLPlatform = "steam"
	RLPlatformPS    RLPlatform = "ps"
	RLPlatformXbox  RLPlatform = "xbox"
)

var AllPlatforms = map[string]struct{}{
	string(RLPlatformEpic):  {},
	string(RLPlatformSteam): {},
	string(RLPlatformPS):    {},
	string(RLPlatformXbox):  {},
}

type TwitchResponseType string

const (
	TwitchResponseTypeMessage TwitchResponseType = "message"
	TwitchResponseTypeReply   TwitchResponseType = "reply"
	TwitchResponseTypeMention TwitchResponseType = "mention"
)

type BotUser struct {
	TwitchUserID    string
	IsAuthenticated bool
}

type EventSubSubscription struct {
	SubscriptionID string
	TwitchUserID   string
	Topic          string
}

type BotCommand struct {
	CommandName            string
	CommandCooldownSeconds int
	MessageFormat          string
	TwitchUserID           string
	TwitchResponseType     TwitchResponseType
	RLPlatform             RLPlatform
	RLUsername             string
}

type CachedCommand struct {
	CommandCooldownSeconds   int
	NextExecutionAllowedTime time.Time
	MessageFormat            string
	TwitchResponseType       TwitchResponseType
	RLPlatform               RLPlatform
	RLUsername               string
}

// CachedAppState This is currently not safe against race conditions and should be reworked if sharding is to be implemented
type CachedAppState struct {
	TwitchAppToken               string
	TwitchAppTokenExpiry         time.Time
	TwitchBotAccountToken        string
	TwitchBotAccountTokenExpiry  time.Time
	TwitchBotAccountRefreshToken string
}
