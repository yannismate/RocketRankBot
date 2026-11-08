package bot

import (
	"context"
	"fmt"
	"text/template"
)

const (
	messageRateLimited         = "Player rank could not be fetched due to rate limiting. Please try again later."
	messageBroadcasterOnly     = "This command can only be executed by the broadcaster."
	messageChannelNameUpdate   = "Your name has changed, the bot has joined your channel under the new name. All commands were transferred."
	messageBotNotJoined        = "The bot is not in your channel."
	messageInvalidPlatform     = "Invalid platform, please use one of the following: epic, steam, ps, xbox."
	messageCommandDoesNotExist = "Command could not be found."
)

var (
	templateMessageNotFound = template.Must(template.New("playerNotFoundTemplate").Parse("Player {{ .PlayerName }} could not be found on {{ .PlayerPlatform }}."))
)

func getMessageInternalErrorWithCtx(ctx context.Context) string {
	return fmt.Sprintf("Internal error occurred while executing the command. Please try again later and reach out if the issue persists (Trace-ID %v).", ctx.Value("trace-id"))
}
