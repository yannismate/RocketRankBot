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
	TwitchUserID string
	TwitchLogin  string
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
