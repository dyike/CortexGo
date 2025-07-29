package agents

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// ChatModel is the shared chat model instance for all agents
var ChatModel model.ChatModel

// InitModel initializes the chat model for agents
func InitModel() error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Printf("OPENAI_API_KEY not set, using default configuration")
		apiKey = "dummy-key" // Fallback for testing
	}

	ctx := context.Background()
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: apiKey,
		Model:  "gpt-3.5-turbo",
	})
	if err != nil {
		return err
	}

	ChatModel = cm
	return nil
}

