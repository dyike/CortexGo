package agents

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

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
