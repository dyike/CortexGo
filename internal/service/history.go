package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

	store, err := getSQLiteStore()
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	ctx := context.Background()
	sessions, err := store.ListSessions(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}

	items := make([]models.HistorySession, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, models.HistorySession{
			SessionID: s.ID,
			Symbol:    s.Symbol,
			TradeDate: s.TradeDate,
			Prompt:    s.Prompt,
			Status:    s.Status,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}

	nextCursor := ""
	if len(sessions) == limit && len(sessions) > 0 {
		nextCursor = strconv.FormatInt(sessions[len(sessions)-1].RowID, 10)
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

	store, err := getSQLiteStore()
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	ctx := context.Background()
	sessionRec, err := store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sessionRec == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	msgRecs, err := store.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]models.HistoryMessage, 0, len(msgRecs))
	for _, m := range msgRecs {
		messages = append(messages, models.HistoryMessage{
			ID:           m.ID,
			Role:         m.Role,
			Agent:        m.Agent,
			Content:      m.Content,
			Status:       m.Status,
			FinishReason: m.FinishReason,
			Seq:          m.Seq,
			CreatedAt:    m.CreatedAt,
			UpdatedAt:    m.UpdatedAt,
		})
	}

	return models.HistoryInfoResponse{
		Session: models.HistorySession{
			SessionID: sessionRec.ID,
			Symbol:    sessionRec.Symbol,
			TradeDate: sessionRec.TradeDate,
			Prompt:    sessionRec.Prompt,
			Status:    sessionRec.Status,
			CreatedAt: sessionRec.CreatedAt,
			UpdatedAt: sessionRec.UpdatedAt,
		},
		Messages: messages,
	}, nil
}
