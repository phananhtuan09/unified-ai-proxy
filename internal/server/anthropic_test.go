package server

import (
	"testing"

	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

func TestParseAnthropicMessagesRequestAcceptsOutputConfig(t *testing.T) {
	req, err := parseAnthropicMessagesRequest([]byte(`{
        "model": "alias",
        "messages": [{"role": "user", "content": [{"type": "text", "text": "hi"}]}],
        "max_tokens": 100,
        "output_config": {"effort": "high"}
    }`))
	if err != nil {
		t.Fatalf("parseAnthropicMessagesRequest() error = %v", err)
	}
	if req.OutputConfig["effort"] != "high" {
		t.Fatalf("output_config = %#v, want effort=high", req.OutputConfig)
	}
}

func TestParseAnthropicMessagesRequestAcceptsThinking(t *testing.T) {
	req, err := parseAnthropicMessagesRequest([]byte(`{
        "model": "alias",
        "messages": [{"role": "user", "content": [{"type": "text", "text": "hi"}]}],
        "max_tokens": 100,
        "thinking": {"type": "enabled", "budget_tokens": 1024}
    }`))
	if err != nil {
		t.Fatalf("parseAnthropicMessagesRequest() error = %v", err)
	}
	if req.Thinking["type"] != "enabled" {
		t.Fatalf("thinking = %#v, want type=enabled", req.Thinking)
	}
}

func TestAnthropicToNormalizedMapsTools(t *testing.T) {
	req, err := parseAnthropicMessagesRequest([]byte(`{
        "model": "alias",
        "tools": [{"name": "bash", "description": "run commands", "input_schema": {"type": "object"}}],
        "messages": [{"role": "user", "content": [{"type": "text", "text": "run it"}]}]
    }`))
	if err != nil {
		t.Fatalf("parseAnthropicMessagesRequest() error = %v", err)
	}
	normalized, err := anthropicToNormalized(req)
	if err != nil {
		t.Fatalf("anthropicToNormalized() error = %v", err)
	}
	if len(normalized.Tools) != 1 || normalized.Tools[0].Name != "bash" {
		t.Fatalf("tools = %#v, want one bash tool", normalized.Tools)
	}
}

func TestAnthropicToNormalizedMapsToolResult(t *testing.T) {
	req, err := parseAnthropicMessagesRequest([]byte(`{
        "model": "alias",
        "messages": [{"role": "user", "content": [{"type": "tool_result", "tool_use_id": "call_1", "content": "done"}]}]
    }`))
	if err != nil {
		t.Fatalf("parseAnthropicMessagesRequest() error = %v", err)
	}
	normalized, err := anthropicToNormalized(req)
	if err != nil {
		t.Fatalf("anthropicToNormalized() error = %v", err)
	}
	if len(normalized.Messages) != 1 || normalized.Messages[0].Role != model.RoleTool || normalized.Messages[0].ToolResult == nil {
		t.Fatalf("messages = %#v, want tool result message", normalized.Messages)
	}
}

func TestAnthropicToNormalizedPreservesEveryToolResult(t *testing.T) {
	req, err := parseAnthropicMessagesRequest([]byte(`{
        "model": "alias",
        "tool_choice": {"type": "auto"},
        "messages": [{"role": "user", "content": [
          {"type": "tool_result", "tool_use_id": "call_1", "content": "first"},
          {"type": "tool_result", "tool_use_id": "call_2", "content": "second"}
        ]}]
    }`))
	if err != nil {
		t.Fatalf("parseAnthropicMessagesRequest() error = %v", err)
	}
	normalized, err := anthropicToNormalized(req)
	if err != nil {
		t.Fatalf("anthropicToNormalized() error = %v", err)
	}
	if len(normalized.Messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(normalized.Messages))
	}
	for i, want := range []struct{ id, content string }{{"call_1", "first"}, {"call_2", "second"}} {
		message := normalized.Messages[i]
		if message.Role != model.RoleTool || message.ToolResult == nil || message.ToolResult.CallID != want.id || message.ToolResult.Content != want.content {
			t.Fatalf("message[%d] = %#v, want tool result %#v", i, message, want)
		}
	}
}

func TestAnthropicToNormalizedAcceptsStringContent(t *testing.T) {
	req, err := parseAnthropicMessagesRequest([]byte(`{
        "model": "alias",
        "messages": [{"role": "user", "content": "hello"}]
    }`))
	if err != nil {
		t.Fatalf("parseAnthropicMessagesRequest() error = %v", err)
	}
	normalized, err := anthropicToNormalized(req)
	if err != nil {
		t.Fatalf("anthropicToNormalized() error = %v", err)
	}
	if normalized.Messages[0].Content != "hello" {
		t.Fatalf("content = %q, want hello", normalized.Messages[0].Content)
	}
}

func TestAnthropicToNormalizedMapsSystemMessage(t *testing.T) {
	req, err := parseAnthropicMessagesRequest([]byte(`{
        "model": "alias",
        "messages": [
          {"role": "system", "content": "follow the policy"},
          {"role": "user", "content": "hello"}
        ]
    }`))
	if err != nil {
		t.Fatalf("parseAnthropicMessagesRequest() error = %v", err)
	}
	normalized, err := anthropicToNormalized(req)
	if err != nil {
		t.Fatalf("anthropicToNormalized() error = %v", err)
	}
	if normalized.System != "follow the policy" {
		t.Fatalf("system = %q, want follow the policy", normalized.System)
	}
	if len(normalized.Messages) != 1 || normalized.Messages[0].Role != model.RoleUser {
		t.Fatalf("messages = %#v, want only user message", normalized.Messages)
	}
}
