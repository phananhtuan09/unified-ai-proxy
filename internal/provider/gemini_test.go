package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

func testGemini(baseURL string) *Gemini {
	cfg := config.ProviderConfig{
		Auth: config.AuthConfig{Method: "api_key"},
		API:  config.APIConfig{BaseURL: baseURL},
		Models: []config.ModelConfig{
			{ID: "gemini-2.5-flash", Upstream: "gemini-2.5-flash"},
		},
	}
	return NewGemini(cfg, 10*time.Second)
}

func testAccount() model.Account {
	return model.Account{Provider: "gemini", Name: "test", APIKey: "AIza-test"}
}

func TestGeminiBuildRequest(t *testing.T) {
	g := testGemini("http://example.com")
	temp := 0.7
	topP := 0.9
	req := &model.ChatRequest{
		Model:         "gemini-2.5-flash",
		System:        "be helpful",
		Temperature:   &temp,
		TopP:          &topP,
		MaxTokens:     100,
		StopSequences: []string{"END"},
		Messages: []model.Message{
			{Role: model.RoleUser, Content: "hi"},
			{Role: model.RoleAssistant, Content: "hello"},
		},
	}

	out := g.buildRequest(req)

	if out.SystemInstruction == nil {
		t.Fatal("expected systemInstruction")
	}
	if out.SystemInstruction.Role != "user" || len(out.SystemInstruction.Parts) != 1 || out.SystemInstruction.Parts[0].Text != "be helpful" {
		t.Fatalf("unexpected systemInstruction: %+v", out.SystemInstruction)
	}
	if len(out.Contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(out.Contents))
	}
	if out.Contents[0].Role != "user" || out.Contents[0].Parts[0].Text != "hi" {
		t.Fatalf("unexpected content[0]: %+v", out.Contents[0])
	}
	if out.Contents[1].Role != "model" || out.Contents[1].Parts[0].Text != "hello" {
		t.Fatalf("assistant should map to model role: %+v", out.Contents[1])
	}
	if out.GenerationConfig == nil {
		t.Fatal("expected generationConfig")
	}
	if out.GenerationConfig.Temperature == nil || *out.GenerationConfig.Temperature != temp {
		t.Fatalf("temperature mismatch: %+v", out.GenerationConfig.Temperature)
	}
	if out.GenerationConfig.TopP == nil || *out.GenerationConfig.TopP != topP {
		t.Fatalf("topP mismatch: %+v", out.GenerationConfig.TopP)
	}
	if out.GenerationConfig.MaxOutputTokens != 100 {
		t.Fatalf("maxOutputTokens mismatch: %d", out.GenerationConfig.MaxOutputTokens)
	}
	if len(out.GenerationConfig.StopSequences) != 1 || out.GenerationConfig.StopSequences[0] != "END" {
		t.Fatalf("stopSequences mismatch: %+v", out.GenerationConfig.StopSequences)
	}
}

func TestGeminiBuildRequestMinimal(t *testing.T) {
	g := testGemini("http://example.com")
	req := &model.ChatRequest{
		Model:    "gemini-2.5-flash",
		Messages: []model.Message{{Role: model.RoleUser, Content: "hi"}},
	}
	out := g.buildRequest(req)
	if out.SystemInstruction != nil {
		t.Fatalf("systemInstruction should be nil: %+v", out.SystemInstruction)
	}
	if out.GenerationConfig != nil {
		t.Fatalf("generationConfig should be nil when empty: %+v", out.GenerationConfig)
	}
}

func TestGeminiChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "AIza-test" {
			t.Errorf("missing or wrong api key header: %q", r.Header.Get("x-goog-api-key"))
		}
		if !strings.HasSuffix(r.URL.Path, ":generateContent") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content":      map[string]any{"role": "model", "parts": []map[string]any{{"text": "Hello"}}},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     5,
				"candidatesTokenCount": 3,
				"totalTokenCount":      8,
			},
		})
	}))
	defer srv.Close()

	g := testGemini(srv.URL)
	resp, err := g.ChatCompletion(context.Background(), testAccount(), &model.ChatRequest{
		Model:     "gemini-2.5-flash",
		MaxTokens: 100,
		Messages:  []model.Message{{Role: model.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Content != "Hello" {
		t.Fatalf("unexpected content %q", resp.Content)
	}
	if resp.StopReason != "STOP" {
		t.Fatalf("unexpected stop reason %q", resp.StopReason)
	}
	if resp.Usage.InputTokens != 5 || resp.Usage.OutputTokens != 3 {
		t.Fatalf("unexpected usage %+v", resp.Usage)
	}
}

func TestGeminiChatCompletionAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "invalid key", "status": "PERMISSION_DENIED"},
		})
	}))
	defer srv.Close()

	g := testGemini(srv.URL)
	_, err := g.ChatCompletion(context.Background(), testAccount(), &model.ChatRequest{
		Model:    "gemini-2.5-flash",
		Messages: []model.Message{{Role: model.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsAuthFailure(err) {
		t.Fatalf("expected auth failure, got %v", err)
	}
}

func TestGeminiChatCompletionEmptyKey(t *testing.T) {
	g := testGemini("http://example.com")
	_, err := g.ChatCompletion(context.Background(), model.Account{Name: "x"}, &model.ChatRequest{
		Model:    "gemini-2.5-flash",
		Messages: []model.Message{{Role: model.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for empty api key")
	}
}

func TestGeminiStreamChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ":streamGenerateContent") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"Hel\"}]},\"finishReason\":null}]}\n\n")
		fl.Flush()
		fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"lo\"}]},\"finishReason\":null}]}\n\n")
		fl.Flush()
		fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":5,\"candidatesTokenCount\":2,\"totalTokenCount\":7}}\n\n")
		fl.Flush()
	}))
	defer srv.Close()

	g := testGemini(srv.URL)
	ch, err := g.StreamChatCompletion(context.Background(), testAccount(), &model.ChatRequest{
		Model:    "gemini-2.5-flash",
		Messages: []model.Message{{Role: model.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("StreamChatCompletion: %v", err)
	}

	var events []model.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d: %+v", len(events), events)
	}
	if events[0].Type != model.StreamMessageStart {
		t.Fatalf("first event should be message_start, got %v", events[0].Type)
	}
	if events[1].Type != model.StreamContentDelta || events[1].Content != "Hel" {
		t.Fatalf("unexpected delta event: %+v", events[1])
	}
	if events[2].Type != model.StreamContentDelta || events[2].Content != "lo" {
		t.Fatalf("unexpected delta event: %+v", events[2])
	}
	if events[3].Type != model.StreamMessageStop || events[3].StopReason != "STOP" {
		t.Fatalf("unexpected stop event: %+v", events[3])
	}
	if events[3].Usage == nil || events[3].Usage.InputTokens != 5 || events[3].Usage.OutputTokens != 2 {
		t.Fatalf("unexpected usage: %+v", events[3].Usage)
	}
}

func TestGeminiValidateAccount(t *testing.T) {
	g := testGemini("http://example.com")
	if err := g.ValidateAccount(context.Background(), model.Account{Name: "x"}); err == nil {
		t.Fatal("expected error for empty key")
	}
	if err := g.ValidateAccount(context.Background(), model.Account{Name: "x", APIKey: "k"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
