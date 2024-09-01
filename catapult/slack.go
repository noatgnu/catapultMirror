// catapult/slack.go
package catapult

import (
	"fmt"
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
		LogWithDatetime(fmt.Sprintf("Failed to send Slack notification: %v", err), false)
	}
}

func TestSlackCredentials(token string) bool {
	client := slack.New(token)
	authTest, err := client.AuthTest()
	if err != nil {
		LogWithDatetime(fmt.Sprintf("Failed to authenticate with Slack: %v", err), false)
		return false
	}
	LogWithDatetime(fmt.Sprintf("Successfully authenticated with Slack. User ID: %v", authTest.UserID), false)
	return true
}
