package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	ecmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/models"
	"github.com/dyike/CortexGo/pkg/utils"
)

type LoggerCallback struct {
	callbacks.HandlerBuilder

	Emit            func(event string, data *models.ChatResp)
	toolCallCacheMu sync.Mutex
	toolCallCache   map[string]toolCallInfo // key: stream message ID
}

type toolCallInfo struct {
	id      string
	name    string
	started bool
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
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

	dataID := msgID
	role := "assistant"
	if strings.TrimSpace(string(msg.Role)) != "" {
		role = string(msg.Role)
	}

	fr := ""
	if msg.ResponseMeta != nil {
		fr = msg.ResponseMeta.FinishReason
	}
	data := &models.ChatResp{
		Agent:         agentName,
		ID:            dataID,
		Role:          role,
		Content:       msg.Content,
		FinishReason:  fr,
		MessageChunks: msg.Content,
	}

	if msg.Role == schema.Tool {
		tcInfo := cb.getToolCallInfo(msgID)
		callID := strings.TrimSpace(msg.ToolCallID)
		if callID == "" {
			callID = tcInfo.id
		}
		if callID == "" {
			callID = msgID + ":tc"
		}
		data.ID = callID + ":result"
		data.ToolCallID = callID
		return cb.pushF(ctx, "tool_call_result", data)
	}

	if len(msg.ToolCalls) > 0 {
		if len(msg.ToolCalls) != 1 {
			return nil
		}

		tcInfo := cb.getToolCallInfo(msgID)
		tcID := strings.TrimSpace(msg.ToolCalls[0].ID)
		fn := strings.TrimSpace(msg.ToolCalls[0].Function.Name)
		if tcID == "" {
			tcID = tcInfo.id
		}
		if fn == "" {
			fn = tcInfo.name
		}
		if tcID == "" {
			tcID = msgID + ":tc"
		}
		cb.rememberToolCall(msgID, tcID, fn)
		argStr := strings.TrimSpace(msg.ToolCalls[0].Function.Arguments)
		argMap := map[string]any{}
		if argStr != "" {
			if err := json.Unmarshal([]byte(argStr), &argMap); err != nil {
				argMap["_raw"] = argStr
			}
		}

		ts := []models.ToolResp{{
			Name: fn,
			Args: argMap,
			Type: "tool_call",
			ID:   tcID,
		}}
		tcs := []models.ToolChunkResp{{
			Name: fn,
			Args: argStr,
			Type: "tool_call_chunk",
			ID:   tcID,
		}}

		if !tcInfo.started {
			_ = cb.pushF(ctx, "message_delta", &models.ChatResp{
				Agent:         agentName,
				ID:            msgID,
				Role:          role,
				Content:       msg.Content,
				FinishReason:  "",
				MessageChunks: msg.Content,
			})
			cb.markToolStarted(msgID, tcID, fn)
			_ = cb.pushF(ctx, "tool_call_started", &models.ChatResp{
				Agent:          agentName,
				ID:             tcID + ":name",
				Role:           role,
				Content:        fn,
				ToolCallID:     tcID,
				ToolCalls:      ts,
				ToolCallChunks: tcs,
			})
		}

		if argStr != "" {
			_ = cb.pushF(ctx, "tool_call_args_delta", &models.ChatResp{
				Agent:          agentName,
				ID:             tcID + ":args",
				Role:           role,
				Content:        argStr,
				ToolCallID:     tcID,
				ToolCalls:      ts,
				ToolCallChunks: tcs,
			})
		}

		if fr != "" {
			_ = cb.pushF(ctx, "tool_call_args_done", &models.ChatResp{
				Agent:          agentName,
				ID:             tcID + ":args",
				Role:           role,
				FinishReason:   fr,
				ToolCallID:     tcID,
				ToolCalls:      ts,
				ToolCallChunks: tcs,
			})
		}
		return nil
	}

	if fr != "" {
		data.FinishReason = fr
		_ = cb.pushF(ctx, "message_delta", data)
		data.Content = ""
		return cb.pushF(ctx, "message_done", data)
	}

	return cb.pushF(ctx, "message_delta", data)
}

func (cb *LoggerCallback) rememberToolCall(streamMsgID, callID, fn string) {
	if strings.TrimSpace(streamMsgID) == "" || strings.TrimSpace(callID) == "" {
		return
	}
	if cb.toolCallCache == nil {
		cb.toolCallCache = make(map[string]toolCallInfo)
	}
	cb.toolCallCacheMu.Lock()
	info := cb.toolCallCache[streamMsgID]
	info.id = callID
	if fn != "" {
		info.name = fn
	}
	cb.toolCallCache[streamMsgID] = info
	cb.toolCallCacheMu.Unlock()
}

func (cb *LoggerCallback) getToolCallInfo(streamMsgID string) toolCallInfo {
	if cb == nil || cb.toolCallCache == nil || strings.TrimSpace(streamMsgID) == "" {
		return toolCallInfo{}
	}
	cb.toolCallCacheMu.Lock()
	defer cb.toolCallCacheMu.Unlock()
	return cb.toolCallCache[streamMsgID]
}

func (cb *LoggerCallback) markToolStarted(streamMsgID, callID, fn string) {
	if cb.toolCallCache == nil {
		cb.toolCallCache = make(map[string]toolCallInfo)
	}
	cb.toolCallCacheMu.Lock()
	info := cb.toolCallCache[streamMsgID]
	info.id = callID
	if fn != "" {
		info.name = fn
	}
	info.started = true
	cb.toolCallCache[streamMsgID] = info
	cb.toolCallCacheMu.Unlock()
}
