package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/dyike/CortexGo/models"
	"github.com/dyike/CortexGo/pkg/utils"
)

type recordKind int

const (
	recordUser recordKind = iota + 1
	recordChunk
	recordError
	recordFinish
	recordSystem
)

type recordEvent struct {
	kind recordKind
	data *models.ChatResp
	err  error
}

type StreamRecorder struct {
	store   *Store
	session models.SessionRecord

	events chan recordEvent
	done   chan struct{}
	once   sync.Once
	wg     sync.WaitGroup

	messageSeq      int
	messageSeen     map[string]struct{}
	messageProgress map[string]int
	messageContent  map[string]string
	hasError        bool
}

func NewStreamRecorder(ctx context.Context, store *Store, session models.SessionRecord) (*StreamRecorder, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	if session.Status == "" {
		session.Status = StatusStreaming
	}
	if err := store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	r := &StreamRecorder{
		store:           store,
		session:         session,
		events:          make(chan recordEvent, 512),
		done:            make(chan struct{}),
		messageSeen:     make(map[string]struct{}),
		messageProgress: make(map[string]int),
		messageContent:  make(map[string]string),
	}

	r.wg.Add(1)
	go r.loop()
	return r, nil
}

func (r *StreamRecorder) loop() {
	defer r.wg.Done()
	ctx := context.Background()
	for ev := range r.events {
		switch ev.kind {
		case recordUser:
			r.handleUser(ctx, ev.data)
		case recordChunk:
			r.handleChunk(ctx, ev.data)
		case recordSystem:
			r.handleSystem(ctx, ev.data)
		case recordError:
			r.handleError(ctx, ev.err, ev.data)
		case recordFinish:
			r.handleFinish(ctx, ev.err)
		}
	}
}

func (r *StreamRecorder) enqueue(ev recordEvent) {
	select {
	case <-r.done:
		return
	case r.events <- ev:
	default:
		go func() {
			select {
			case <-r.done:
			case r.events <- ev:
			}
		}()
	}
}

func (r *StreamRecorder) RecordUserPrompt(prompt string) {
	if strings.TrimSpace(prompt) == "" {
		return
	}
	r.enqueue(recordEvent{
		kind: recordUser,
		data: &models.ChatResp{
			ID:      utils.RandStr(16),
			Role:    "user",
			Content: prompt,
		},
	})
}

func (r *StreamRecorder) HandleStreamEvent(event string, data *models.ChatResp) {
	switch event {
	case "message_chunk", "tool_calls", "tool_call_chunks", "tool_call_result":
		r.enqueue(recordEvent{kind: recordChunk, data: data})
	case "run_start":
		r.enqueue(recordEvent{kind: recordSystem, data: data})
	case "error":
		err := error(nil)
		if data != nil && strings.TrimSpace(data.Content) != "" {
			err = errors.New(data.Content)
		}
		r.enqueue(recordEvent{kind: recordError, data: data, err: err})
	}
}

func (r *StreamRecorder) Finish(err error) {
	if err != nil {
		r.enqueue(recordEvent{kind: recordError, err: err})
	}
	r.enqueue(recordEvent{kind: recordFinish, err: err})
	r.Close()
}

func (r *StreamRecorder) Close() {
	r.once.Do(func() {
		close(r.done)
		close(r.events)
		r.wg.Wait()
	})
}

func (r *StreamRecorder) handleUser(ctx context.Context, data *models.ChatResp) {
	if data == nil {
		return
	}
	msgID := data.ID
	if strings.TrimSpace(msgID) == "" {
		msgID = utils.RandStr(20)
	}
	role := data.Role
	if strings.TrimSpace(role) == "" {
		role = "user"
	}
	if err := r.ensureMessage(ctx, msgID, role, data.Agent, StatusDone); err != nil {
		log.Printf("record user message: %v", err)
		return
	}
	if strings.TrimSpace(data.Content) != "" {
		if err := r.store.AppendMessageContent(ctx, msgID, data.Content); err != nil {
			log.Printf("append user content: %v", err)
		}
		r.messageContent[msgID] += data.Content
		r.messageProgress[msgID] = len(r.messageContent[msgID])
	}
	_ = r.store.MarkMessageStatus(ctx, msgID, StatusDone, data.FinishReason)
}

func (r *StreamRecorder) handleSystem(ctx context.Context, data *models.ChatResp) {
	if data == nil {
		return
	}
	msgID := data.ID
	if strings.TrimSpace(msgID) == "" {
		msgID = utils.RandStr(20)
	}
	role := data.Role
	if strings.TrimSpace(role) == "" {
		role = "system"
	}
	if err := r.ensureMessage(ctx, msgID, role, data.Agent, StatusDone); err != nil {
		log.Printf("record system message: %v", err)
		return
	}
	if strings.TrimSpace(data.Content) != "" {
		if err := r.store.AppendMessageContent(ctx, msgID, data.Content); err != nil {
			log.Printf("append system content: %v", err)
		}
		r.messageContent[msgID] += data.Content
		r.messageProgress[msgID] = len(r.messageContent[msgID])
	}
	_ = r.store.MarkMessageStatus(ctx, msgID, StatusDone, data.FinishReason)
}

func (r *StreamRecorder) handleChunk(ctx context.Context, data *models.ChatResp) {
	if data == nil || strings.TrimSpace(data.ID) == "" {
		return
	}
	role := data.Role
	if strings.TrimSpace(role) == "" {
		role = "assistant"
	}
	if err := r.ensureMessage(ctx, data.ID, role, data.Agent, StatusStreaming); err != nil {
		log.Printf("record message: %v", err)
		return
	}

	raw := data.Content
	if raw == "" {
		raw = toolCallContent(data)
	}
	if raw != "" {
		existing := r.messageContent[data.ID]
		delta := raw

		switch {
		case strings.HasPrefix(raw, existing):
			delta = raw[len(existing):]
			r.messageContent[data.ID] = raw
		case existing == "":
			r.messageContent[data.ID] = raw
		default:
			r.messageContent[data.ID] = existing + delta
		}

		if delta != "" {
			if err := r.store.AppendMessageContent(ctx, data.ID, delta); err != nil {
				log.Printf("append chunk: %v", err)
			}
		}
		r.messageProgress[data.ID] = len(r.messageContent[data.ID])
	}

	if strings.TrimSpace(data.FinishReason) != "" {
		if err := r.store.MarkMessageStatus(ctx, data.ID, StatusDone, data.FinishReason); err != nil {
			log.Printf("finish message: %v", err)
		}
	}
}

func (r *StreamRecorder) handleError(ctx context.Context, err error, data *models.ChatResp) {
	r.hasError = true
	_ = r.store.UpdateSessionStatus(ctx, r.session.ID, StatusError)
	_ = r.store.FinalizeOpenMessages(ctx, r.session.ID, StatusError, "error")

	msgID := ""
	role := "system"
	content := ""

	if data != nil {
		msgID = strings.TrimSpace(data.ID)
		role = strings.TrimSpace(data.Role)
		content = data.Content
		if strings.TrimSpace(role) == "" {
			role = "system"
		}
	}
	if msgID == "" {
		msgID = utils.RandStr(20)
	}
	if content == "" && err != nil {
		content = err.Error()
	}

	if ensureErr := r.ensureMessage(ctx, msgID, role, "", StatusError); ensureErr != nil {
		log.Printf("record error message: %v", ensureErr)
		return
	}
	if strings.TrimSpace(content) != "" {
		if appErr := r.store.AppendMessageContent(ctx, msgID, content); appErr != nil {
			log.Printf("append error content: %v", appErr)
		}
	}
	_ = r.store.MarkMessageStatus(ctx, msgID, StatusError, "error")
}

func (r *StreamRecorder) handleFinish(ctx context.Context, err error) {
	status := StatusDone
	finishReason := ""
	if err != nil || r.hasError {
		status = StatusError
		finishReason = "error"
	}
	_ = r.store.FinalizeOpenMessages(ctx, r.session.ID, status, finishReason)
	_ = r.store.UpdateSessionStatus(ctx, r.session.ID, status)
}

func (r *StreamRecorder) ensureMessage(ctx context.Context, msgID, role, agent, status string) error {
	if msgID == "" {
		return fmt.Errorf("message id is required")
	}
	if _, ok := r.messageSeen[msgID]; ok {
		return nil
	}
	if status == "" {
		status = StatusStreaming
	}
	r.messageSeq++
	if err := r.store.InsertMessage(ctx, models.MessageRecord{
		ID:        msgID,
		SessionID: r.session.ID,
		Role:      role,
		Agent:     agent,
		Status:    status,
		Seq:       r.messageSeq,
	}); err != nil {
		return err
	}
	r.messageSeen[msgID] = struct{}{}
	r.messageProgress[msgID] = 0
	return nil
}

func toolCallContent(data *models.ChatResp) string {
	if data == nil {
		return ""
	}

	if len(data.ToolCallChunks) > 0 {
		tc := data.ToolCallChunks[0]
		args := strings.TrimSpace(tc.Args)
		if args != "" {
			name := strings.TrimSpace(tc.Name)
			if name != "" {
				return fmt.Sprintf("[tool_call:%s] %s", name, args)
			}
			return args
		}
	}

	if len(data.ToolCalls) > 0 && len(data.ToolCalls[0].Args) > 0 {
		if argJSON, err := json.Marshal(data.ToolCalls[0].Args); err == nil {
			name := strings.TrimSpace(data.ToolCalls[0].Name)
			if name != "" {
				return fmt.Sprintf("[tool_call:%s] %s", name, string(argJSON))
			}
			return string(argJSON)
		}
	}

	return ""
}
