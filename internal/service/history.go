package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dyike/CortexGo/internal/storage"
	"github.com/dyike/CortexGo/models"
)

// GetAgentHistory 从 sqlite 中按 rowid 倒序分页列出历史会话（不含消息内容）
func GetAgentHistory(paramsJson string) (any, error) {
	var params models.HistoryParams
	if strings.TrimSpace(paramsJson) != "" {
		if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var cursor int64
	if strings.TrimSpace(params.Cursor) != "" {
		val, err := strconv.ParseInt(params.Cursor, 10, 64)
		if err != nil || val < 0 {
			return nil, fmt.Errorf("invalid cursor")
		}
		cursor = val
	}

	store, err := storage.GetSQLiteStore()
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	ctx := context.Background()
	query := strings.TrimSpace(params.Query)
	symbol := strings.TrimSpace(params.Symbol)
	tradeDate := strings.TrimSpace(params.TradeDate)
	if query != "" {
		symbol = ""
		tradeDate = ""
	}

	sessions, err := store.ListSessions(ctx, cursor, limit, symbol, tradeDate, query)
	if err != nil {
		return nil, err
	}

	items := make([]models.HistorySession, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, models.HistorySession{
			SessionID: strconv.FormatInt(s.Id, 10),
			Symbol:    s.Symbol,
			TradeDate: s.TradeDate,
			Prompt:    s.Prompt,
			Status:    s.Status,
			CreatedAt: formatTime(s.CreatedAt),
			UpdatedAt: formatTime(s.UpdatedAt),
		})
	}

	nextCursor := ""
	if len(sessions) == limit && len(sessions) > 0 {
		nextCursor = strconv.FormatInt(sessions[len(sessions)-1].Id, 10)
	}

	return models.HistoryListResponse{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}, nil
}

// GetHistoryInfo 根据 session_id 读取会话详情及消息内容
func GetHistoryInfo(paramsJson string) (any, error) {
	var params models.HistoryInfoParams
	if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	sessionInt, err := strconv.ParseInt(sessionID, 10, 64)
	if err != nil || sessionInt <= 0 {
		return nil, fmt.Errorf("invalid session_id")
	}

	store, err := storage.GetSQLiteStore()
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	ctx := context.Background()
	sessionRec, err := store.GetSession(ctx, sessionInt)
	if err != nil {
		return nil, err
	}
	if sessionRec == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	msgRecs, err := store.ListMessages(ctx, sessionInt)
	if err != nil {
		return nil, err
	}

	messages := make([]models.HistoryMessage, 0, len(msgRecs))
	for _, m := range msgRecs {
		messages = append(messages, models.HistoryMessage{
			ID:           strconv.FormatInt(m.Id, 10),
			Role:         m.Role,
			Agent:        m.Agent,
			Content:      m.Content,
			ToolCallId:   m.ToolCallId,
			ToolName:     m.ToolName,
			ToolCalls:    parseToolCalls(m.ToolCalls),
			Status:       m.Status,
			FinishReason: m.FinishReason,
			Seq:          m.Seq,
			CreatedAt:    formatTime(m.CreatedAt),
			UpdatedAt:    formatTime(m.UpdatedAt),
		})
	}

	return models.HistoryInfoResponse{
		Session: models.HistorySession{
			SessionID: strconv.FormatInt(sessionRec.Id, 10),
			Symbol:    sessionRec.Symbol,
			TradeDate: sessionRec.TradeDate,
			Prompt:    sessionRec.Prompt,
			Status:    sessionRec.Status,
			CreatedAt: formatTime(sessionRec.CreatedAt),
			UpdatedAt: formatTime(sessionRec.UpdatedAt),
		},
		Messages: messages,
	}, nil
}

// DeleteHistory 根据 session_id 删除会话及其消息
func DeleteHistory(paramsJson string) (any, error) {
	var params models.HistoryDeleteParams
	if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	sessionInt, err := strconv.ParseInt(sessionID, 10, 64)
	if err != nil || sessionInt <= 0 {
		return nil, fmt.Errorf("invalid session_id")
	}

	store, err := storage.GetSQLiteStore()
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	ctx := context.Background()
	if err := store.DeleteSession(ctx, sessionInt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, err
	}

	return models.HistoryDeleteResponse{
		SessionID: sessionID,
		Deleted:   true,
	}, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseToolCalls(raw string) []*models.ToolCall {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var calls []*models.ToolCall
	if err := json.Unmarshal([]byte(raw), &calls); err != nil {
		fmt.Printf("parse tool_calls err=%v\n", err)
		return nil
	}
	return calls
}
