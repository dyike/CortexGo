package agents

import (
	"context"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
)

var (
	ChatModel *openai.ChatModel
	chatMu    sync.Mutex
)

func InitChatModel(ctx context.Context, cfg *config.Config) error {
	if ChatModel != nil {
		return nil
	}

	chatMu.Lock()
	defer chatMu.Unlock()

	if ChatModel != nil {
		return nil
	}

	maxTokens := 8192
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:   "https://api.deepseek.com/v1",
		APIKey:    cfg.DeepSeekAPIKey,
		Model:     "deepseek-chat",
		MaxTokens: &maxTokens,
	})
	if err != nil {
		return err
	}
	ChatModel = chatModel
	return nil
}

func ToolCallChecker(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()
	for {
		msg, err := sr.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				return false, nil
			}
			return false, err
		}
		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
}
