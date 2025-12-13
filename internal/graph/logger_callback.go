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
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/models"
	"github.com/dyike/CortexGo/pkg/utils"
)

type toolCallInfo struct {
	id               string
	name             string
	argumentsBuilder strings.Builder
}

type LoggerCallback struct {
	callbacks.HandlerBuilder

	Emit func(event string, data *models.ChatResp)

	// 运行时状态追踪
	currentContent strings.Builder
	toolCalls      map[string]*toolCallInfo // key: tool_call ID
	stateLock      sync.Mutex
}

func NewLoggerCallback(emit func(event string, data *models.ChatResp)) *LoggerCallback {
	return &LoggerCallback{
		Emit:      emit,
		toolCalls: make(map[string]*toolCallInfo),
	}
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

// User prompt and System prompt
func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close() // remember to close the stream in defer
	return ctx
}

// AI返回的结果
func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {

	// 在开始处理新流之前，清空状态以避免数据混淆
	cb.stateLock.Lock()
	cb.currentContent.Reset()
	cb.toolCalls = make(map[string]*toolCallInfo)
	cb.stateLock.Unlock()

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
				cb.flushCurrentAssistantMessage(true)
				break
			}
			if err != nil {
				fmt.Println("=========[OnEndStream]recv_error=========", err)
				cb.flushCurrentAssistantMessage(true)
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
			default:
			}
		}

	}()
	return ctx
}

func (cb *LoggerCallback) pushMsg(ctx context.Context, msgID string, msg *schema.Message) error {
	if msg == nil {
		return nil
	}

	cb.stateLock.Lock()
	defer cb.stateLock.Unlock()

	if cb.Emit == nil {
		return nil
	}

	// --- 1. 处理工具执行结果 (Role: tool) ---
	// 这种消息通常是完整的，直接发送用于持久化
	if msg.Role == schema.Tool {
		cb.flushCurrentAssistantMessage(true)

		var toolMsgs []struct {
			Role       string `json:"role"`
			Content    string `json:"content"`
			ToolCallId string `json:"tool_call_id"`
			ToolName   string `json:"tool_name"`
		}
		if err := json.Unmarshal([]byte(msg.Content), &toolMsgs); err == nil && len(toolMsgs) > 0 {
			cb.Emit("tool_result_final", &models.ChatResp{
				Role:       toolMsgs[0].Role,
				Content:    toolMsgs[0].Content,
				ToolCallId: toolMsgs[0].ToolCallId,
			})
		}
		return nil
	}
	// --- 2. 处理 LLM 助手流 (Role: assistant) ---
	// A. 累积文本内容
	if msg.Content != "" {
		cb.currentContent.WriteString(msg.Content)

		cb.Emit("text_chunk", &models.ChatResp{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// B. 累积工具调用参数
	if msg.ToolCalls != nil {
		for _, tc := range msg.ToolCalls {
			info, exists := cb.toolCalls[tc.ID]

			if !exists && tc.ID != "" {
				// 第一次看到带有 ID 的工具调用，初始化
				info = &toolCallInfo{id: tc.ID}
				cb.toolCalls[tc.ID] = info
			} else if !exists {
				// 如果 ID 为空，尝试附加到当前唯一的工具调用上
				if len(cb.toolCalls) == 1 {
					for _, activeInfo := range cb.toolCalls {
						info = activeInfo
						break
					}
				} else {
					continue // 无法确定累积到哪个，跳过
				}
			}

			// 确保找到了 info
			if info == nil {
				continue
			}

			// 更新 name
			if tc.Function.Name != "" {
				info.name = tc.Function.Name
			}

			// 累积 arguments
			if tc.Function.Arguments != "" {
				info.argumentsBuilder.WriteString(tc.Function.Arguments)

				// 实时流式回调：发送工具调用参数片段（可选，用于显示工具调用过程）
				if cb.Emit != nil {
					// 只发送包含 ID 的片段，没有ID的参数片段不具意义
					if tc.ID != "" || info.id != "" {
						cb.Emit("tool_call_chunk", &models.ChatResp{
							Role: string(msg.Role),
							ToolCalls: []*models.ToolCall{
								&models.ToolCall{
									Id:   info.id,
									Type: tc.Type,
									Function: struct {
										Name      string "json:\"name\""
										Arguments string "json:\"arguments\""
									}{
										Name:      info.name,
										Arguments: tc.Function.Arguments,
									},
								},
							},
						})
					}
				}
			}
		}
	}
	// C. 检查结束标志并发送最终消息
	if msg.ResponseMeta != nil &&
		(msg.ResponseMeta.FinishReason == "stop" || msg.ResponseMeta.FinishReason == "tool_calls") {
		cb.flushCurrentAssistantMessage(false) // 正常结束，进行落地
	}

	// agentName := ""
	// _ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
	// 	agentName = state.Goto
	// 	return nil
	// })

	return nil
}

func (cb *LoggerCallback) flushCurrentAssistantMessage(force bool) {

	// 聚合的文本内容或工具调用请求是本次 Assistant 消息的有效载荷
	hasContent := cb.currentContent.Len() > 0
	hasToolCalls := len(cb.toolCalls) > 0

	if !hasContent && !hasToolCalls && !force {
		return // 没有内容，且不是强制落地，直接返回
	}

	// 1. 聚合工具调用请求
	var finalToolCalls []*models.ToolCall
	if hasToolCalls {
		for id, info := range cb.toolCalls {
			// 确保参数是有效的 JSON，虽然聚合后的字符串可能不是严格有效的，但我们尽力而为
			tc := &models.ToolCall{
				Id:   id,
				Type: "function", // 假设类型
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      info.name,
					Arguments: info.argumentsBuilder.String(), // 完整的参数 JSON 字符串
				},
			}
			finalToolCalls = append(finalToolCalls, tc)
		}
	}

	// 2. 构建最终消息
	finalMsg := &models.ChatResp{
		Role:      "assistant",
		Content:   cb.currentContent.String(),
		ToolCalls: finalToolCalls,
	}

	// 3. 触发持久化回调
	if cb.Emit != nil {
		if hasToolCalls {
			// 优先发送工具调用请求的最终消息 (Persistence)
			cb.Emit("tool_call_request_final", finalMsg)
		} else if hasContent {
			// 发送纯文本的最终消息 (Persistence)
			cb.Emit("text_final", finalMsg)
		} else if force {
			// 强制落地，但没有内容，可以发送一个空消息（根据业务需求决定是否需要）
			// 为了简化，我们只在有内容或工具调用时发送
		}
	}

	// 4. 清理状态，准备接收下一轮消息
	cb.currentContent.Reset()
	cb.toolCalls = make(map[string]*toolCallInfo)
}
