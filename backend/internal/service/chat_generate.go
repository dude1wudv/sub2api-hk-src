package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

func chatPublicError(err error) map[string]any {
	var app *infraerrors.ApplicationError
	if !errors.As(err, &app) {
		app = ErrChatStream
	}
	return map[string]any{"kind": "http", "code": app.Reason, "status": int(app.Code), "message": app.Message}
}

func (s *ChatService) persistRejectedGeneration(record *ChatSessionRecord, assistantID string, err error) {
	message, decodeErr := ChatAssistantJSON(record.Snapshot, assistantID)
	if decodeErr != nil || message["status"] != "pending" {
		return
	}
	message["status"] = "error"
	message["error"] = chatPublicError(err)
	encoded, decodeErr := json.Marshal(message)
	if decodeErr != nil {
		return
	}
	raw, _, decodeErr := MergeChatAssistant(record, assistantID, encoded, "")
	if decodeErr != nil {
		return
	}
	snapshot, decodeErr := DecodeChatSnapshot(record.ID, raw)
	if decodeErr != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// This CAS also rejects an intervening generation or another device's save.
	_, _ = s.repo.SaveSession(ctx, record.UserID, snapshot, record.Revision)
}

func (s *ChatService) Generate(ctx context.Context, userID int64, sessionID uuid.UUID, req ChatGenerateRequest, clientIP string, out http.ResponseWriter) (resultErr error) {
	if req.PresetID == uuid.Nil || req.RequestID == uuid.Nil || req.ExpectedRevision == nil || *req.ExpectedRevision < 0 || req.AssistantMessageID == "" {
		return ErrChatInvalid
	}
	switch req.Operation {
	case "send", "regenerate", "continue":
	default:
		return ErrChatInvalid
	}
	record, err := s.repo.GetSession(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if record.Revision != *req.ExpectedRevision {
		return ErrChatRevisionConflict
	}
	snapshot, err := DecodeChatSnapshot(sessionID, record.Snapshot)
	if err != nil {
		return err
	}
	path := snapshot.VisibleMessages()
	if len(path) < 2 || path[len(path)-1].ID != req.AssistantMessageID || path[len(path)-1].Role != "assistant" || path[len(path)-2].Role != "user" {
		return ErrChatInvalid
	}
	target := path[len(path)-1]
	if target.Status != "pending" {
		return ErrChatInvalid
	}
	claimed := false
	defer func() {
		if resultErr != nil && !claimed {
			s.persistRejectedGeneration(record, req.AssistantMessageID, resultErr)
		}
	}()
	preset, model, _, err := s.models.ResolvePreset(ctx, userID, req.PresetID)
	if err != nil {
		return err
	}
	preferences, _, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return err
	}
	var prefs ChatPreferences
	if json.Unmarshal(preferences, &prefs) != nil {
		return ErrChatInvalid
	}
	options := ChatRunOptions{Thinking: target.Thinking, Search: target.Search, Location: prefs.Location, Locale: prefs.Locale, Transport: model.Capabilities.Transport}
	if req.Operation == "continue" {
		previous, err := s.repo.LastGeneration(ctx, userID, sessionID, target.ID)
		if err != nil {
			return ErrChatProviderContext
		}
		if previous.GroupID != preset.GroupID || previous.Model != preset.Model || previous.Status == "running" || !target.Prefix {
			return ErrChatProviderContext
		}
		if json.Unmarshal(previous.Options, &options) != nil || options.Transport != model.Capabilities.Transport {
			return ErrChatProviderContext
		}
	} else if len(target.Blocks) != 0 {
		return ErrChatInvalid
	}
	if options.Search && !model.Capabilities.Search {
		return ErrChatCapability
	}
	if model.Capabilities.NativeDeepSeekExtensions && options.LatestReminder == "" {
		options.LatestReminder = "Current time (UTC): " + time.Now().UTC().Format(time.RFC3339)
		if options.Locale != "" {
			options.LatestReminder += ". Language preference: " + options.Locale
		}
		if options.Location != nil {
			location, _ := json.Marshal(options.Location)
			options.LatestReminder += ". User-provided location: " + string(location)
		}
	}
	prepared, err := s.buildChatRequest(ctx, userID, snapshot, preset, model, options, req.Operation)
	if err != nil {
		return err
	}
	options = prepared.options
	key, err := s.credentials.Ensure(ctx, userID, preset.GroupID)
	if err != nil {
		return err
	}
	message, err := ChatAssistantJSON(record.Snapshot, target.ID)
	if err != nil {
		return err
	}
	message["model"], message["groupId"], message["presetId"] = preset.Model, preset.GroupID, preset.ID.String()
	message["thinking"], message["search"], message["chatOptions"] = options.Thinking, options.Search, options
	if options.LatestReminder != "" {
		message["latestReminder"] = options.LatestReminder
	}
	message["status"] = "streaming"
	delete(message, "error")
	delete(message, "stopReason")
	delete(message, "stopDetails")
	initial, err := json.Marshal(message)
	if err != nil {
		return err
	}
	rawOptions, err := json.Marshal(options)
	if err != nil {
		return err
	}
	run := &ChatRunRecord{ID: uuid.New(), UserID: userID, SessionID: sessionID, AssistantMessageID: target.ID, PresetID: preset.ID, GroupID: preset.GroupID, Model: preset.Model, RequestID: req.RequestID, Operation: req.Operation, Status: "running", InstanceID: s.instanceID, Options: rawOptions}
	claimedRecord, err := s.repo.ClaimGeneration(ctx, run, *req.ExpectedRevision, initial)
	if err != nil {
		return err
	}
	record = claimedRecord
	claimed = true
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	active, err := s.registerRun(run, cancel)
	if err != nil {
		finalCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = s.repo.StopGeneration(finalCtx, userID, sessionID, run.ID)
		return err
	}
	defer s.finishRun(run.ID, active)
	state := newChatReplyAccumulator(message)
	state.title = chatFallbackTitle(path)
	var streamMu sync.Mutex
	dirty, writeFailed := false, false
	lastSave := time.Now()
	emit := func(event map[string]any) error {
		streamMu.Lock()
		defer streamMu.Unlock()
		if err := state.accept(event); err != nil {
			return err
		}
		if event["type"] != "ping" {
			dirty = true
		}
		if dirty && time.Since(lastSave) >= time.Second {
			raw, err := json.Marshal(state.message)
			if err != nil {
				return err
			}
			saved, err := s.repo.SaveGeneration(runCtx, run, raw, state.title, "running")
			if err != nil {
				return err
			}
			record = saved
			lastSave = time.Now()
			dirty = false
		}
		if err := flushSSEJSON(out, chatString(event["type"]), event); err != nil {
			writeFailed = true
			cancel()
			return err
		}
		return nil
	}
	out.Header().Set("Content-Type", "text/event-stream")
	out.Header().Set("Cache-Control", "private, no-store")
	out.Header().Set("X-Accel-Buffering", "no")
	out.WriteHeader(http.StatusOK)
	runErr := emit(map[string]any{"type": "chat_run", "run_id": run.ID.String()})
	heartCtx, stopHeart := context.WithCancel(runCtx)
	heartDone := make(chan struct{})
	heartErr := make(chan error, 1)
	go func() {
		defer close(heartDone)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-heartCtx.Done():
				return
			case <-ticker.C:
				renewal, done := context.WithTimeout(heartCtx, 5*time.Second)
				err := s.repo.RenewGeneration(renewal, run)
				done()
				if err != nil {
					if heartCtx.Err() == nil {
						heartErr <- err
						cancel()
					}
					return
				}
				if err = emit(map[string]any{"type": "ping"}); err != nil {
					heartErr <- err
					cancel()
					return
				}
			}
		}
	}()
	if runErr == nil && len(prepared.documents) > 0 {
		runErr = s.prepareChatDocuments(runCtx, run, clientIP, prepared.documents)
		if runErr == nil {
			prepared, runErr = s.buildChatRequest(runCtx, userID, snapshot, preset, model, options, req.Operation)
			if runErr == nil && len(prepared.documents) > 0 {
				runErr = ErrChatParseBusy
			}
		}
	}
	if runErr == nil {
		runErr = s.executeChatRequest(runCtx, key, run, clientIP, prepared, state, emit)
	}
	stopHeart()
	<-heartDone
	select {
	case err := <-heartErr:
		runErr = err
	default:
	}
	status := "done"
	if writeFailed || ctx.Err() != nil {
		status = "stopped"
	} else if runErr != nil {
		status = "error"
	}
	state.message["status"] = status
	if status == "error" {
		state.message["error"] = chatPublicError(runErr)
	}
	terminal, marshalErr := json.Marshal(state.message)
	finalCtx, finalCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finalCancel()
	var saved *ChatSessionRecord
	saveErr := marshalErr
	if saveErr == nil {
		saved, saveErr = s.repo.SaveGeneration(finalCtx, run, terminal, state.title, status)
	}
	if saveErr != nil {
		// Never declare completion when its canonical state was not committed.
		_ = flushSSEJSON(out, "error", map[string]any{"type": "error", "error": chatPublicError(saveErr)})
		return nil
	}
	if status == "done" {
		_ = flushSSEJSON(out, "message_stop", map[string]any{"type": "message_stop"})
	}
	_ = flushSSEJSON(out, "chat_saved", map[string]any{"type": "chat_saved", "revision": saved.Revision})
	if status != "done" {
		if runErr == nil {
			runErr = ErrChatLeaseLost
		}
		_ = flushSSEJSON(out, "error", map[string]any{"type": "error", "error": chatPublicError(runErr)})
	}
	return nil
}

func chatFallbackTitle(messages []ChatMessage) string {
	for _, message := range messages {
		if message.Role == "user" {
			text := []rune(strings.TrimSpace(message.Content))
			if len(text) > 40 {
				text = text[:40]
			}
			return string(text)
		}
	}
	return ""
}

type chatReplyAccumulator struct {
	message map[string]any
	blocks  []any
	title   string
	size    int
}

func newChatReplyAccumulator(message map[string]any) *chatReplyAccumulator {
	blocks, _ := message["blocks"].([]any)
	if blocks == nil {
		blocks = []any{}
	}
	message["blocks"] = blocks
	return &chatReplyAccumulator{message: message, blocks: blocks}
}
func (a *chatReplyAccumulator) accept(event map[string]any) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	a.size += len(encoded)
	if a.size > ChatMaxSessionBytes {
		return ErrChatTooLarge
	}
	kind := chatString(event["type"])
	switch kind {
	case "message_start":
		if usage := chatObject(chatObject(event["message"])["usage"]); usage != nil {
			a.message["usage"] = usage
		}
	case "message_delta":
		delta := chatObject(event["delta"])
		if reason := chatString(delta["stop_reason"]); reason != "" {
			a.message["stopReason"] = reason
		}
		if details := delta["stop_details"]; details != nil {
			a.message["stopDetails"] = details
		}
		if title := strings.TrimSpace(chatString(delta["title"])); title != "" {
			a.title = title
		}
		if usage := chatObject(event["usage"]); usage != nil {
			current := chatObject(a.message["usage"])
			if current == nil {
				current = map[string]any{}
			}
			for key, value := range usage {
				current[key] = value
			}
			a.message["usage"] = current
		}
	case "content_block_start":
		index := chatInteger(event["index"])
		if index != len(a.blocks) {
			return ErrChatStream
		}
		block := chatObject(event["content_block"])
		if block == nil {
			return ErrChatStream
		}
		if block["type"] == "thinking" {
			block["startedAt"] = time.Now().UnixMilli()
		}
		a.blocks = append(a.blocks, block)
		a.message["blocks"] = a.blocks
	case "content_block_delta", "content_block_stop":
		index := chatInteger(event["index"])
		if index < 0 || index >= len(a.blocks) {
			return ErrChatStream
		}
		block := chatObject(a.blocks[index])
		if block == nil {
			return ErrChatStream
		}
		if kind == "content_block_stop" {
			if block["type"] == "thinking" {
				block["endedAt"] = time.Now().UnixMilli()
			}
			return nil
		}
		delta := chatObject(event["delta"])
		switch delta["type"] {
		case "text_delta":
			block["text"] = chatString(block["text"]) + chatString(delta["text"])
		case "thinking_delta":
			block["thinking"] = chatString(block["thinking"]) + chatString(delta["thinking"])
		case "signature_delta":
			block["signature"] = chatString(block["signature"]) + chatString(delta["signature"])
		case "citations_delta":
			citations, _ := block["citations"].([]any)
			block["citations"] = append(citations, delta["citation"])
		case "input_json_delta":
			return fmt.Errorf("unexpected client tool delta: %w", ErrChatStream)
		}
	}
	return nil
}
