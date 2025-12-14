package storage

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dyike/CortexGo/models"
)

// StreamRecorder 将流式事件写入 SQLite，补充 session 元信息。
type StreamRecorder struct {
	ctx       context.Context
	store     *Store
	sessionID string
}

func NewStreamRecorder(ctx context.Context, store *Store, session models.SessionRecord) (*StreamRecorder, error) {
	if store == nil {
		return nil, fmt.Errorf("store is nil")
	}
	session.ID = strings.TrimSpace(session.ID)
	if session.ID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if err := store.UpsertSession(ctx, session); err != nil {
		return nil, err
	}
	return &StreamRecorder{
		ctx:       ctx,
		store:     store,
		sessionID: session.ID,
	}, nil
}

// RecordUserPrompt 在会话开始时记录用户提示。
func (r *StreamRecorder) RecordUserPrompt(prompt string) {
	if r == nil || r.store == nil {
		return
	}
	if strings.TrimSpace(prompt) == "" {
		return
	}
	_ = r.store.SaveMessage(r.ctx, r.sessionID, "user_prompt_final", &models.ChatResp{
		Role:    "user",
		Content: prompt,
	})
}

// HandleStreamEvent 处理流式事件，并在 *_final 阶段落地。
func (r *StreamRecorder) HandleStreamEvent(event string, data *models.ChatResp) {
	if r == nil || r.store == nil || data == nil {
		return
	}
	if err := r.store.SaveMessage(r.ctx, r.sessionID, event, data); err != nil {
		log.Printf("record stream event %s: %v", event, err)
	}
}

// Finish 更新 session 状态。
func (r *StreamRecorder) Finish(err error) {
	if r == nil || r.store == nil {
		return
	}
	status := StatusDone
	if err != nil {
		status = StatusError
	}
	if updateErr := r.store.UpdateSessionStatus(r.ctx, r.sessionID, status); updateErr != nil {
		log.Printf("update session status: %v", updateErr)
	}
}
