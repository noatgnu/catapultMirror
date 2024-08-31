// catapult/slack.go
package catapult

import (
	"github.com/slack-go/slack"
	"os"
)

var slackClient *slack.Client
var slackChannelID string

func InitSlack(config Configurations) {
	token := config.SlackToken
	if token == "" {
		token = os.Getenv("SLACK_TOKEN")
	}

	channelID := config.SlackChannelID
	if channelID == "" {
		channelID = os.Getenv("SLACK_CHANNEL_ID")
	}

	if token != "" {
		slackClient = slack.New(token)
		slackChannelID = channelID
	}
}

func sendSlackNotification(message string) {
	if slackClient == nil || slackChannelID == "" {
		return
	}

	_, _, err := slackClient.PostMessage(
		slackChannelID,
		slack.MsgOptionText(message, false),
	)
	if err != nil {
		LogWithDatetime("Failed to send Slack notification:", err)
	}
}

func TestSlackCredentials(token string) bool {
	client := slack.New(token)
	authTest, err := client.AuthTest()
	if err != nil {
		LogWithDatetime("Failed to authenticate with Slack:", err)
		return false
	}
	LogWithDatetime("Successfully authenticated with Slack. User ID:", authTest.UserID)
	return true
}
