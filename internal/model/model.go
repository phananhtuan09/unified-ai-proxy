package model

import "time"

// Role is a normalized message role understood by every provider.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleDeveloper Role = "developer"
	RoleTool      Role = "tool"
)

// ToolCall is a single tool invocation requested by the model.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResult is the output of a tool invocation returned to the model.
type ToolResult struct {
	CallID  string `json:"call_id"`
	Content string `json:"content"`
}

// Tool describes a function the model may call.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// Message is a normalized chat message. Text content lives in Content; tool
// calls and tool results are carried in the dedicated fields.
type Message struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ChatRequest is the normalized internal request used across providers.
type ChatRequest struct {
	Provider      string         `json:"provider"`
	Model         string         `json:"model"`
	Messages      []Message      `json:"messages"`
	System        string         `json:"system,omitempty"`
	Tools         []Tool         `json:"tools,omitempty"`
	Stream        bool           `json:"stream"`
	Temperature   *float64       `json:"temperature,omitempty"`
	TopP          *float64       `json:"top_p,omitempty"`
	MaxTokens     int            `json:"max_tokens"`
	StopSequences []string       `json:"stop_sequences,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// Usage holds token accounting for a single completion.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ChatResponse is the normalized non-streaming response.
type ChatResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	Content    string `json:"content"`
	StopReason string `json:"stop_reason,omitempty"`
	Usage      Usage  `json:"usage"`
}

// StreamEventType enumerates the normalized streaming event kinds.
type StreamEventType string

const (
	StreamMessageStart StreamEventType = "message_start"
	StreamContentDelta StreamEventType = "content_delta"
	StreamToolCall     StreamEventType = "tool_call"
	StreamMessageStop  StreamEventType = "message_stop"
	StreamError        StreamEventType = "error"
)

// StreamEvent is a normalized streaming event.
type StreamEvent struct {
	Type       StreamEventType
	ID         string
	Model      string
	Content    string
	ToolCall   *ToolCall
	StopReason string
	Usage      *Usage
	Error      error
}

// Model is a configured upstream model alias.
type Model struct {
	ID       string `json:"id"`
	Upstream string `json:"upstream"`
	Provider string `json:"provider"`
}

// Account identifies a single upstream account configuration.
type Account struct {
	Provider  string `json:"provider"`
	Name      string `json:"name"`
	TokenFile string `json:"token_file"`
	APIKey    string `json:"api_key,omitempty"`
}

// TokenSet is the persisted OAuth token schema.
type TokenSet struct {
	Provider     string    `json:"provider"`
	Account      string    `json:"account"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsExpired reports whether the access token has expired (with 30s skew).
func (t *TokenSet) IsExpired(now time.Time) bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return now.Add(30 * time.Second).After(t.ExpiresAt)
}

// NeedsRefresh reports whether the token has a refresh token and is expired.
func (t *TokenSet) NeedsRefresh(now time.Time) bool {
	return t.RefreshToken != "" && t.IsExpired(now)
}
