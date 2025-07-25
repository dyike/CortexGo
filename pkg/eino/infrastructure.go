package eino

import (
	"context"
	"os"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

var (
	ChatModel model.Model
)

func InitModel() error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "your_api_key_here"
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	modelConfig := &openaiModel.ChatModelConfig{
		Model:      "gpt-4",
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Temperature: 0.7,
	}

	chatModel, err := openaiModel.NewChatModel(context.Background(), modelConfig)
	if err != nil {
		return err
	}

	ChatModel = chatModel
	return nil
}

func GetModelInstance() model.Model {
	return ChatModel
}

func CreateSystemMessage(content string) *schema.Message {
	return schema.SystemMessage(content)
}

func CreateUserMessage(content string) *schema.Message {
	return schema.UserMessage(content)
}

func CreateAssistantMessage(content string) *schema.Message {
	return schema.AssistantMessage(content)
}