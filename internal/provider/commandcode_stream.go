package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

type ndjsonEvent struct {
	Type         string          `json:"type"`
	ID           string          `json:"id"`
	ToolCallID   string          `json:"toolCallId"`
	ToolName     string          `json:"toolName"`
	Input        json.RawMessage `json:"input"`
	Text         json.RawMessage `json:"text"`
	Delta        json.RawMessage `json:"delta"`
	FinishReason json.RawMessage `json:"finishReason"`
	Usage        json.RawMessage `json:"usage"`
	TotalUsage   json.RawMessage `json:"totalUsage"`
	Error        json.RawMessage `json:"error"`
	Message      json.RawMessage `json:"message"`
}

type ndjsonUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

func (c *CommandCode) parseNDJSON(ctx context.Context, r io.Reader, upstreamModel, apiKey string, out chan<- model.StreamEvent) {
	defer close(out)
	defer func() {
		if closer, ok := r.(io.Closer); ok {
			_ = closer.Close()
		}
	}()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxNDJSONLine)
	emitter := &commandCodeStreamEmitter{out: out, model: upstreamModel, apiKey: apiKey}
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var ev ndjsonEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if !emitter.handle(ev) {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			emitter.fail(errors.New("upstream NDJSON line exceeds limit"))
		} else {
			emitter.fail(fmt.Errorf("upstream NDJSON read error: %w", err))
		}
		return
	}
	if !emitter.stopSent {
		emitter.fail(errors.New("stream ended before terminal event"))
	}
}

type commandCodeStreamEmitter struct {
	out               chan<- model.StreamEvent
	model, apiKey     string
	started, stopSent bool
	toolInput         map[string]*strings.Builder
	toolName          map[string]string
}

func (e *commandCodeStreamEmitter) start() {
	if !e.started {
		e.out <- model.StreamEvent{Type: model.StreamMessageStart, ID: "chatcmpl-" + mustRandomHex(16), Model: e.model}
		e.started = true
	}
}
func (e *commandCodeStreamEmitter) fail(err error) {
	e.out <- model.StreamEvent{Type: model.StreamError, Error: fmt.Errorf("%s", sanitizeMessage(err.Error(), e.apiKey))}
}
func (e *commandCodeStreamEmitter) handle(ev ndjsonEvent) bool {
	switch ev.Type {
	case "start":
		e.start()
	case "text-delta":
		text, err := decodeTextDelta(ev.Text, ev.Delta)
		if err != nil {
			e.fail(err)
			return false
		}
		if text != "" {
			e.start()
			e.out <- model.StreamEvent{Type: model.StreamContentDelta, Content: text}
		}
	case "tool-input-start":
		e.start()
		id := ev.ToolCallID
		if id == "" {
			id = ev.ID
		}
		if id == "" {
			e.fail(errors.New("tool-input-start event missing tool call id"))
			return false
		}
		if e.toolInput == nil {
			e.toolInput = make(map[string]*strings.Builder)
			e.toolName = make(map[string]string)
		}
		e.toolInput[id] = new(strings.Builder)
		e.toolName[id] = ev.ToolName
	case "tool-input-delta":
		id := ev.ToolCallID
		if id == "" {
			id = ev.ID
		}
		if b, ok := e.toolInput[id]; ok {
			var delta string
			if err := json.Unmarshal(ev.Delta, &delta); err != nil {
				e.fail(fmt.Errorf("tool-input-delta event has non-string delta"))
				return false
			}
			b.WriteString(delta)
		}
	case "tool-input-end":
		id := ev.ToolCallID
		if id == "" {
			id = ev.ID
		}
		if b, ok := e.toolInput[id]; ok {
			e.out <- model.StreamEvent{Type: model.StreamToolCall, ToolCall: &model.ToolCall{ID: id, Name: e.toolName[id], Arguments: b.String()}}
		}
	case "tool-call":
		id := ev.ToolCallID
		if id == "" {
			id = ev.ID
		}
		if _, ok := e.toolInput[id]; !ok {
			e.start()
			args := ""
			if len(ev.Input) > 0 && string(ev.Input) != "null" {
				args = string(ev.Input)
			}
			e.out <- model.StreamEvent{Type: model.StreamToolCall, ToolCall: &model.ToolCall{ID: id, Name: ev.ToolName, Arguments: args}}
		}
	case "finish-step", "finish":
		raw := ev.FinishReason
		usageRaw := ev.Usage
		if ev.Type == "finish" {
			usageRaw = ev.TotalUsage
		}
		reason, ok := decodeStringField(raw)
		if !ok {
			e.fail(fmt.Errorf("%s event has non-string finishReason", ev.Type))
			return false
		}
		usage, ok := decodeUsage(usageRaw)
		if !ok {
			e.fail(fmt.Errorf("%s event has malformed usage", ev.Type))
			return false
		}
		e.start()
		if !e.stopSent {
			e.stopSent = true
			e.out <- model.StreamEvent{Type: model.StreamMessageStop, StopReason: mapCommandCodeStopReason(reason), Usage: usage}
		}
	case "error":
		e.fail(errors.New(decodeErrorMessage(ev.Error, ev.Message)))
		return false
	}
	return true
}
func mapCommandCodeStopReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return reason
	}
}
func decodeTextDelta(textRaw, deltaRaw json.RawMessage) (string, error) {
	for _, raw := range []json.RawMessage{textRaw, deltaRaw} {
		if len(raw) > 0 && string(raw) != "null" {
			var s string
			if err := json.Unmarshal(raw, &s); err != nil {
				return "", errors.New("text-delta event has non-string text field")
			}
			return s, nil
		}
	}
	return "", nil
}
func decodeStringField(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", true
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return s, true
}
func decodeUsage(raw json.RawMessage) (*model.Usage, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, true
	}
	var u ndjsonUsage
	if json.Unmarshal(raw, &u) != nil {
		return nil, false
	}
	return &model.Usage{InputTokens: u.InputTokens, OutputTokens: u.OutputTokens}, true
}
func decodeErrorMessage(errorRaw, messageRaw json.RawMessage) string {
	for _, raw := range []json.RawMessage{messageRaw, errorRaw} {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && s != "" {
			return s
		}
		var obj map[string]any
		if json.Unmarshal(raw, &obj) == nil {
			if m, ok := obj["message"].(string); ok && m != "" {
				return m
			}
		}
	}
	return "unknown upstream error"
}
