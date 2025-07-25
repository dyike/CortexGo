package eino

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type MockChatModel struct{}

func (m *MockChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("Mock response from trading agent", []schema.ToolCall{
		{
			ID:   "mock_call",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "execute_trade",
				Arguments: `{"action": "buy", "quantity": 100, "reasoning": "Mock trading decision"}`,
			},
		},
	}), nil
}

func (m *MockChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

var ChatModel model.BaseChatModel

func InitModel() error {
	ChatModel = &MockChatModel{}
	return nil
}

func CreateSystemMessage(content string) *schema.Message {
	return schema.SystemMessage(content)
}

func CreateUserMessage(content string) *schema.Message {
	return schema.UserMessage(content)
}

func CreateAssistantMessage(content string) *schema.Message {
	return schema.AssistantMessage(content, nil)
}