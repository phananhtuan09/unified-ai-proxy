package provider

import (
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

type commandCodeContentBlock struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ToolCallID string          `json:"toolCallId,omitempty"`
	ToolName   string          `json:"toolName,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	Content    string          `json:"content,omitempty"`
}

type commandCodeMessage struct {
	Role    string                    `json:"role"`
	Content []commandCodeContentBlock `json:"content"`
}

type commandCodeTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type commandCodeConfig struct {
	WorkingDir    string   `json:"workingDir"`
	Date          string   `json:"date"`
	Environment   string   `json:"environment"`
	Structure     []string `json:"structure"`
	IsGitRepo     bool     `json:"isGitRepo"`
	CurrentBranch string   `json:"currentBranch"`
	MainBranch    string   `json:"mainBranch"`
	GitStatus     string   `json:"gitStatus"`
	RecentCommits []string `json:"recentCommits"`
}

type commandCodeParams struct {
	Model       string               `json:"model"`
	Messages    []commandCodeMessage `json:"messages"`
	System      string               `json:"system,omitempty"`
	Tools       []commandCodeTool    `json:"tools,omitempty"`
	Stream      bool                 `json:"stream"`
	MaxTokens   int                  `json:"max_tokens"`
	Temperature *float64             `json:"temperature,omitempty"`
	TopP        *float64             `json:"top_p,omitempty"`
}

type commandCodeRequest struct {
	ThreadID string            `json:"threadId"`
	Memory   string            `json:"memory"`
	Config   commandCodeConfig `json:"config"`
	Params   commandCodeParams `json:"params"`
}

func (c *CommandCode) buildRequest(req *model.ChatRequest, sessionID string) *commandCodeRequest {
	params := commandCodeParams{Model: req.Model, System: req.System, Stream: true, MaxTokens: req.MaxTokens, Temperature: req.Temperature, TopP: req.TopP}
	for _, t := range req.Tools {
		params.Tools = append(params.Tools, commandCodeTool{Name: t.Name, Description: t.Description, InputSchema: t.Parameters})
	}
	for _, m := range req.Messages {
		role := string(m.Role)
		if role == "system" || role == "developer" {
			continue
		}
		switch role {
		case "tool":
			if m.ToolResult != nil {
				params.Messages = append(params.Messages, commandCodeMessage{Role: "user", Content: []commandCodeContentBlock{{
					Type:      "tool_result",
					ToolUseID: m.ToolResult.CallID,
					Content:   m.ToolResult.Content,
				}}})
			}
		case "assistant":
			blocks := []commandCodeContentBlock{}
			if m.Content != "" {
				blocks = append(blocks, commandCodeContentBlock{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, commandCodeContentBlock{Type: "tool-call", ToolCallID: tc.ID, ToolName: tc.Name, Input: json.RawMessage(tc.Arguments)})
			}
			if len(params.Messages) > 0 && params.Messages[len(params.Messages)-1].Role == "assistant" {
				params.Messages[len(params.Messages)-1].Content = append(params.Messages[len(params.Messages)-1].Content, blocks...)
			} else {
				params.Messages = append(params.Messages, commandCodeMessage{Role: "assistant", Content: blocks})
			}
		default:
			params.Messages = append(params.Messages, commandCodeMessage{Role: role, Content: []commandCodeContentBlock{{Type: "text", Text: m.Content}}})
		}
	}
	return &commandCodeRequest{ThreadID: sessionID, Config: c.buildConfig(), Params: params}
}

func (c *CommandCode) buildConfig() commandCodeConfig {
	wd, _ := os.Getwd()
	return commandCodeConfig{WorkingDir: wd, Date: time.Now().UTC().Format("2006-01-02"), Environment: runtime.GOOS, Structure: []string{}, RecentCommits: []string{}}
}
