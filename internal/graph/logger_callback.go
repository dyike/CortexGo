package graph

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	ecmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/utils"
	"github.com/dyike/CortexGo/models"
)

type LoggerCallback struct {
	callbacks.HandlerBuilder

	Emit func(event string, data *models.ChatResp)
}

func (cb *LoggerCallback) pushF(ctx context.Context, event string, data *models.ChatResp) error {
	if cb.Emit != nil && data != nil {
		cb.Emit(event, data)
	}
	return nil
}

func (cb *LoggerCallback) pushMsg(ctx context.Context, msgID string, msg *schema.Message) error {
	if msg == nil {
		return nil
	}

	agentName := ""
	_ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		agentName = state.Goto
		return nil
	})

	fr := ""
	if msg.ResponseMeta != nil {
		fr = msg.ResponseMeta.FinishReason
	}
	data := &models.ChatResp{
		Agent:         agentName,
		ID:            msgID,
		Role:          "assistant",
		Content:       msg.Content,
		FinishReason:  fr,
		MessageChunks: msg.Content,
	}

	if msg.Role == schema.Tool {
		data.ToolCallID = msg.ToolCallID
		return cb.pushF(ctx, "tool_call_result", data)
	}

	if len(msg.ToolCalls) > 0 {
		event := "tool_call_chunks"
		if len(msg.ToolCalls) != 1 {

			return nil
		}

		ts := []models.ToolResp{}
		tcs := []models.ToolChunkResp{}
		fn := msg.ToolCalls[0].Function.Name
		if len(fn) > 0 {
			event = "tool_calls"
			if strings.HasSuffix(fn, "search") {
				fn = "web_search"
			}
			ts = append(ts, models.ToolResp{
				Name: fn,
				Args: map[string]interface{}{},
				Type: "tool_call",
				ID:   msg.ToolCalls[0].ID,
			})
		}
		tcs = append(tcs, models.ToolChunkResp{
			Name: fn,
			Args: msg.ToolCalls[0].Function.Arguments,
			Type: "tool_call_chunk",
			ID:   msg.ToolCalls[0].ID,
		})
		data.ToolCalls = ts
		data.ToolCallChunks = tcs
		return cb.pushF(ctx, event, data)
	}
	return cb.pushF(ctx, "message_chunk", data)
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if inputStr, ok := input.(string); ok {
		if cb.Emit != nil {
			cb.Emit("run_start", &models.ChatResp{
				Role:    "system",
				Content: fmt.Sprintf("[OnStart] %s", inputStr),
			})
		}
	}
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	// fmt.Println("=========[OnEnd]=========", info.Name, "|", info.Component, "|", info.Type)
	// outputStr, _ := json.MarshalIndent(output, "", "  ")
	// fmt.Println(string(outputStr))
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Println("=========[OnError]=========")
	fmt.Println(err)
	if cb.Emit != nil {
		cb.Emit("error", &models.ChatResp{
			Role:    "system",
			Content: err.Error(),
		})
	}
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	msgID := utils.RandStr(20)
	go func() {
		defer output.Close() // remember to close the stream in defer
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("=========[OnEndStream]panic_recover=========", err)
			}
		}()
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Println("=========[OnEndStream]recv_error=========", err)
				return
			}

			switch v := frame.(type) {
			case *schema.Message:
				_ = cb.pushMsg(ctx, msgID, v)
			case *ecmodel.CallbackOutput:
				_ = cb.pushMsg(ctx, msgID, v.Message)
			case []*schema.Message:
				for _, m := range v {
					_ = cb.pushMsg(ctx, msgID, m)
				}
			//case string:
			//	ilog.EventInfo(ctx, "frame_type", "type", "str", "v", v)
			default:
				//ilog.EventInfo(ctx, "frame_type", "type", "unknown", "v", v)
			}
		}

	}()
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	return ctx
}
