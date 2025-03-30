package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/sirupsen/logrus"
)

type MattermostClient struct {
	client  *model.Client4
	botUser *model.User
}

func NewMattermostClient() (*MattermostClient, error) {
	mmURL := os.Getenv("MATTERMOST_URL")
	botToken := os.Getenv("MATTERMOST_TOKEN")

	client := model.NewAPIv4Client(mmURL)
	client.SetToken(botToken)

	botUser, resp, err := client.GetMe("")
	if err != nil {
		logrus.Errorf("Failed to authenticate bot: %v (status: %d)", err, resp.StatusCode)
		return nil, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("authentication failed with HTTP %d", resp.StatusCode)
	}

	return &MattermostClient{
		client:  client,
		botUser: botUser,
	}, nil
}

func handleIncomingMessage(event *model.WebSocketEvent, mmClient *MattermostClient, ttClient *TarantoolClient) {
	rawPost := event.GetData()["post"]
	postJSON, ok := rawPost.(string)
	if !ok {
		logrus.Warn("Invalid post format in WebSocket event")
		return
	}

	var post model.Post
	if err := json.Unmarshal([]byte(postJSON), &post); err != nil {
		logrus.Errorf("Failed to parse post: %v", err)
		return
	}

	if post.UserId == mmClient.botUser.Id {
		return
	}

	if !strings.HasPrefix(post.Message, "/") {
		return
	}

	channelID := post.ChannelId
	if channelID == "" {
		logrus.Warn("Missing channel ID in post")
		return
	}

	parts := strings.Fields(post.Message)
	if len(parts) == 0 {
		return
	}

	cmd := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	response, err := handleCommand(cmd, args, ttClient)
	if err != nil {
		logrus.Errorf("Command error: %v", err)
		response = fmt.Sprintf("**Error:** %s", err.Error())
	}

	if err := mmClient.SendResponse(response, channelID); err != nil {
		logrus.Errorf("Failed to send response: %v", err)
		_ = mmClient.SendResponse("Failed to send response. Please try again later.", channelID)
	}
}

func (mc *MattermostClient) SendResponse(message, channelID string) error {
	post := &model.Post{
		ChannelId: channelID,
		Message:   message,
	}

	createdPost, resp, err := mc.client.CreatePost(post)
	if err != nil {
		return fmt.Errorf("API request failed: %w (HTTP %d)", err, resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API returned HTTP %d: %s",
			resp.StatusCode,
			http.StatusText(resp.StatusCode))
	}

	logrus.Infof("Sent message to channel %s (post ID: %s)", channelID, createdPost.Id)
	return nil
}
