package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

func TestOpenAIChatRequestMatrix(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "missing model", body: `{"messages":[{"role":"user","content":"hi"}]}`, want: "model is required"},
		{name: "missing messages", body: `{"model":"alias","messages":[]}`, want: "messages must not be empty"},
		{name: "unsupported field", body: `{"model":"alias","messages":[],"response_format":{}}`, want: "unsupported field: response_format"},
		{name: "invalid role", body: `{"model":"alias","messages":[{"role":"invalid","content":"hi"}]}`, want: "invalid message role"},
		{name: "invalid stop", body: `{"model":"alias","messages":[{"role":"user","content":"hi"}],"stop":1}`, want: "stop must be a string or array"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := parseOpenAIChatRequest([]byte(tc.body))
			if err != nil && tc.name == "unsupported field" {
				if err.Error() != tc.want {
					t.Fatalf("got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			_, err = openAIToNormalized(req)
			if err == nil || !containsError(err, tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestOpenAIChatRequestAcceptsOMPToolFields(t *testing.T) {
	body := `{
		"model":"alias",
		"messages":[
			{"role":"user","content":"run pwd"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"bash","arguments":"{\"command\":\"pwd\"}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"/tmp"}
		],
		"tools":[{"type":"function","function":{"name":"bash","description":"run a command","parameters":{"type":"object"}}}],
		"max_completion_tokens":123,
		"preserve_thinking":true,
		"chat_template_kwargs":{"enable_thinking":true},
		"stream_options":{"include_usage":true},
		"store":false
	}`

	req, err := parseOpenAIChatRequest([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	normalized, err := openAIToNormalized(req)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if normalized.MaxTokens != 123 || len(normalized.Tools) != 1 || normalized.Tools[0].Name != "bash" {
		t.Fatalf("unexpected normalized request: %+v", normalized)
	}
	if len(normalized.Messages) != 3 || normalized.Messages[1].ToolCalls[0].Name != "bash" || normalized.Messages[2].ToolResult.CallID != "call_1" {
		t.Fatalf("tool messages were not normalized: %+v", normalized.Messages)
	}
}

func TestWriteOpenAIChatResponseIncludesToolCalls(t *testing.T) {
	recorder := httptest.NewRecorder()
	c := newTestContext(recorder)
	writeOpenAIChatResponse(c, "cc-model", &model.ChatResponse{
		ToolCalls: []model.ToolCall{{ID: "call_1", Name: "bash", Arguments: `{"command":"pwd"}`}},
	})

	var body struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Choices) != 1 || body.Choices[0].FinishReason != "tool_calls" || len(body.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	call := body.Choices[0].Message.ToolCalls[0]
	if call.ID != "call_1" || call.Type != "function" || call.Function.Name != "bash" || call.Function.Arguments != `{"command":"pwd"}` {
		t.Fatalf("unexpected tool call: %+v", call)
	}
}

func TestWriteResponsesResponseIncludesToolCalls(t *testing.T) {
	recorder := httptest.NewRecorder()
	c := newTestContext(recorder)
	writeResponsesResponse(c, "cc-model", "cc-model", &model.ChatResponse{
		ToolCalls: []model.ToolCall{{ID: "call_1", Name: "bash", Arguments: `{"command":"pwd"}`}},
	})

	var body struct {
		Output []struct {
			Type      string `json:"type"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Output) != 1 || body.Output[0].Type != "function_call" || body.Output[0].CallID != "call_1" || body.Output[0].Name != "bash" || body.Output[0].Arguments != `{"command":"pwd"}` {
		t.Fatalf("unexpected response output: %+v", body.Output)
	}
}

func newTestContext(w *httptest.ResponseRecorder) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestOpenAIContentAndResponsesNormalization(t *testing.T) {
	content, err := decodeOpenAIContent(json.RawMessage(`[{"type":"text","text":"hello"},{"type":"input_text","text":" world"}]`))
	if err != nil || content != "hello world" {
		t.Fatalf("content=%q err=%v", content, err)
	}
	_, _, normalized, err := parseOpenAIResponsesRequest([]byte(`{"model":"alias","input":"hello"}`))
	if err != nil || len(normalized.Messages) != 1 || normalized.Messages[0].Content != "hello" {
		t.Fatalf("normalized=%+v err=%v", normalized, err)
	}
}

func TestResponsesRequestToolCallRoundTrip(t *testing.T) {
	body := `{
		"model":"alias",
		"tools":[{"name":"bash","description":"run","parameters":{"type":"object"}}],
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"run pwd"}]},
			{"type":"function_call","call_id":"call_1","name":"bash","arguments":"{\"command\":\"pwd\"}"},
			{"type":"function_call_output","call_id":"call_1","output":"/tmp"}
		]
	}`
	_, _, normalized, err := parseOpenAIResponsesRequest([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(normalized.Tools) != 1 || normalized.Tools[0].Name != "bash" {
		t.Fatalf("tools mismatch: %+v", normalized.Tools)
	}
	if len(normalized.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %+v", normalized.Messages)
	}
	user := normalized.Messages[0]
	if user.Role != "user" || user.Content != "run pwd" {
		t.Fatalf("user message mismatch: %+v", user)
	}
	assistant := normalized.Messages[1]
	if assistant.Role != "assistant" || len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].Name != "bash" {
		t.Fatalf("assistant tool call mismatch: %+v", assistant)
	}
	tool := normalized.Messages[2]
	if tool.Role != "tool" || tool.ToolResult == nil || tool.ToolResult.CallID != "call_1" || tool.ToolResult.Content != "/tmp" {
		t.Fatalf("tool result mismatch: %+v", tool)
	}
}

func containsError(err error, want string) bool {
	return len(err.Error()) >= len(want) && stringContains(err.Error(), want)
}
func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
