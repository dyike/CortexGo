package agents

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/dyike/CortexGo/internal/config"
)

var (
	ChatModel *openai.ChatModel
)

func InitChatModel(ctx context.Context, cfg *config.Config) {
	maxTokens := 8192
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:   "https://api.deepseek.com/v1",
		APIKey:    cfg.DeepSeekAPIKey,
		Model:     "deepseek-chat",
		MaxTokens: &maxTokens,
	})
	if err != nil {
		panic(err)
	}
	ChatModel = chatModel
}
