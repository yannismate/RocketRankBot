package bot

import "text/template"

const (
	messageInternalError     = "Internal error occurred while executing the command. Please try again later and reach out if the issue persists."
	messageRateLimited       = "Player rank could not be fetched due to rate limiting. Please try again later."
	messageBroadcasterOnly   = "This command can only be executed by the broadcaster."
	messageChannelNameUpdate = "Your name has changed, the bot has joined your channel under the new name. All commands were transferred."
	messageBotNotJoined      = "The bot is not in your channel."
	messageInvalidPlatform   = "Invalid platform, please use one of the following: epic, steam, ps, xbox."
)

var (
	templateMessageNotFound = template.Must(template.New("playerNotFoundTemplate").Parse("Player {{ .PlayerName }} could not be found on {{ .PlayerPlatform }}."))
)
