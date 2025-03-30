package main

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func handleCreateVote(args []string, ttClient *TarantoolClient) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: /create_vote <question> [option1] [option2]...")
	}

	question := args[0]
	options := args[1:]

	if len(options) < 1 {
		return "", fmt.Errorf("you must provide at least one option")
	}

	idStr := generateUniqueID()

	err := ttClient.CreateVote(idStr, "creator", question, options)
	if err != nil {
		return "", fmt.Errorf("failed to create vote: %w", err)
	}

	return fmt.Sprintf("Vote created with ID: `%s`\nOptions: %s", idStr, strings.Join(options, ", ")), nil
}

func handleVote(args []string, ttClient *TarantoolClient) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("usage: /vote <ID> <option>")
	}

	id, option := args[0], args[1]
	err := ttClient.RecordVote(id, option)
	if err != nil {
		return "", fmt.Errorf("failed to record vote: %w", err)
	}
	return fmt.Sprintf("Vote for `%s` recorded successfully!", option), nil
}

func handleResults(args []string, ttClient *TarantoolClient) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("usage: /results <ID>")
	}

	id := args[0]
	results, err := ttClient.GetResults(id)
	if err != nil {
		return "", fmt.Errorf("failed to fetch results: %w", err)
	}

	if results == nil {
		return "", fmt.Errorf("vote ID `%s` not found", id)
	}

	var resultStr strings.Builder
	resultStr.WriteString(fmt.Sprintf("**Results for vote %s**\n", id))
	for option, count := range results {
		resultStr.WriteString(fmt.Sprintf("- %s: %d\n", option, count))
	}
	return resultStr.String(), nil
}

func handleEndVote(args []string, ttClient *TarantoolClient) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("usage: /end_vote <ID>")
	}

	id := args[0]
	err := ttClient.EndVote(id)
	if err != nil {
		return "", fmt.Errorf("failed to end vote: %w", err)
	}
	return fmt.Sprintf("Vote `%s` has been ended", id), nil
}

func handleDeleteVote(args []string, ttClient *TarantoolClient) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("usage: /delete_vote <ID>")
	}

	id := args[0]
	err := ttClient.DeleteVote(id)
	if err != nil {
		return "", fmt.Errorf("failed to delete vote: %w", err)
	}
	return fmt.Sprintf("Vote `%s` deleted successfully", id), nil
}

func generateUniqueID() string {
	return uuid.New().String()
}

func handleCommand(cmd string, args []string, ttClient *TarantoolClient) (string, error) {
	switch cmd {
	case "create_vote":
		return handleCreateVote(args, ttClient)
	case "vote":
		return handleVote(args, ttClient)
	case "results":
		return handleResults(args, ttClient)
	case "end_vote":
		return handleEndVote(args, ttClient)
	case "delete_vote":
		return handleDeleteVote(args, ttClient)
	default:
		return "", fmt.Errorf("unknown command. Available commands: create_vote, vote, results, end_vote, delete_vote")
	}
}
