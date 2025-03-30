package main

import (
	"log"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Info("Starting Voting Bot...")

	mmClient, err := NewMattermostClient()
	if err != nil {
		log.Fatalf("Failed to initialize Mattermost client: %v", err)
	}

	ttClient, err := NewTarantoolClient()
	if err != nil {
		log.Fatalf("Failed to connect to Tarantool: %v", err)
	}

	StartBot(mmClient, ttClient)
}

func StartBot(mmClient *MattermostClient, ttClient *TarantoolClient) {
	logrus.Info("Bot is now listening for commands via WebSocket...")

	wsURL := strings.Replace(mmClient.client.URL, "http", "ws", 1) + "/api/v4/websocket"

	wsClient, err := model.NewWebSocketClient4(wsURL, mmClient.client.AuthToken)
	if err != nil {
		logrus.Fatalf("Failed to create WebSocket client: %v", err)
	}
	defer wsClient.Close()

	if err = wsClient.Connect(); err != nil {
		logrus.Fatalf("WebSocket connection failed: %v", err)
	}
	logrus.Info("WebSocket connection established. Listening for events...")

	wsClient.Listen()

	for {
		select {
		case event, ok := <-wsClient.EventChannel:
			if !ok {
				logrus.Info("Connection closed, attempting to reconnect...")
				if err := wsClient.Connect(); err != nil {
					logrus.Errorf("Reconnect failed: %v", err)
					return
				}
				wsClient.Listen()
				continue
			}

			if event != nil && event.EventType() == model.WebsocketEventPosted {
				handleIncomingMessage(event, mmClient, ttClient)
			}
		}
	}
}
