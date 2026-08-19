package provider

import "github.com/tuanp-github/unified-ai-proxy/internal/model"

// commandCodeModels là registry cứng các model của Command Code.
// Nguồn: https://commandcode.ai/docs/reference/cli/models và /provider/v1/models
// Alias theo convention: cc-<slug> (slug = upstream ID viết thường, thay / và . bằng -)
var commandCodeModels = []model.Model{
	// Alibaba (Qwen)
	{ID: "cc-qwen3-6-max-preview", Upstream: "Qwen/Qwen3.6-Max-Preview", Provider: "command_code"},
	{ID: "cc-qwen3-6-plus", Upstream: "Qwen/Qwen3.6-Plus", Provider: "command_code"},
	{ID: "cc-qwen3-7-flash", Upstream: "Qwen/Qwen3.7-Flash", Provider: "command_code"},
	{ID: "cc-qwen3-7-max", Upstream: "Qwen/Qwen3.7-Max", Provider: "command_code"},
	{ID: "cc-qwen3-7-plus", Upstream: "Qwen/Qwen3.7-Plus", Provider: "command_code"},
	{ID: "cc-qwen3-8-max", Upstream: "Qwen/Qwen3.8-Max", Provider: "command_code"},

	// Anthropic (Claude)
	{ID: "cc-claude-fable-5", Upstream: "claude-fable-5", Provider: "command_code"},
	{ID: "cc-claude-haiku-4-5", Upstream: "claude-haiku-4-5", Provider: "command_code"},
	{ID: "cc-claude-opus-4-7", Upstream: "claude-opus-4-7", Provider: "command_code"},
	{ID: "cc-claude-opus-4-8", Upstream: "claude-opus-4-8", Provider: "command_code"},
	{ID: "cc-claude-opus-5", Upstream: "claude-opus-5", Provider: "command_code"},
	{ID: "cc-claude-sonnet-4-6", Upstream: "claude-sonnet-4-6", Provider: "command_code"},
	{ID: "cc-claude-sonnet-5", Upstream: "claude-sonnet-5", Provider: "command_code"},

	// DeepSeek
	{ID: "cc-deepseek-v4-flash", Upstream: "deepseek/deepseek-v4-flash", Provider: "command_code"},
	{ID: "cc-deepseek-v4-pro", Upstream: "deepseek/deepseek-v4-pro", Provider: "command_code"},

	// Google (Gemini)
	{ID: "cc-gemini-3-1-flash-lite", Upstream: "google/gemini-3.1-flash-lite", Provider: "command_code"},
	{ID: "cc-gemini-3-5-flash", Upstream: "google/gemini-3.5-flash", Provider: "command_code"},
	{ID: "cc-gemini-3-5-flash-lite", Upstream: "google/gemini-3.5-flash-lite", Provider: "command_code"},
	{ID: "cc-gemini-3-6-flash", Upstream: "google/gemini-3.6-flash", Provider: "command_code"},
	{ID: "cc-gemini-3-7-flash", Upstream: "google/gemini-3.7-flash", Provider: "command_code"},

	// Meta (Muse Spark)
	{ID: "cc-muse-spark-1-1", Upstream: "meta/muse-spark-1.1", Provider: "command_code"},
	{ID: "cc-muse-spark-1-2", Upstream: "meta/muse-spark-1.2", Provider: "command_code"},
	{ID: "cc-muse-spark-1-2-contributor", Upstream: "meta/muse-spark-1.2-contributor", Provider: "command_code"},

	// MiniMax
	{ID: "cc-minimax-m2-5", Upstream: "MiniMaxAI/MiniMax-M2.5", Provider: "command_code"},
	{ID: "cc-minimax-m2-7", Upstream: "MiniMaxAI/MiniMax-M2.7", Provider: "command_code"},
	{ID: "cc-minimax-m3", Upstream: "MiniMaxAI/MiniMax-M3", Provider: "command_code"},

	// Moonshot AI (Kimi)
	{ID: "cc-kimi-k2-5", Upstream: "moonshotai/Kimi-K2.5", Provider: "command_code"},
	{ID: "cc-kimi-k2-6", Upstream: "moonshotai/Kimi-K2.6", Provider: "command_code"},
	{ID: "cc-kimi-k2-7-code", Upstream: "moonshotai/Kimi-K2.7-Code", Provider: "command_code"},
	{ID: "cc-kimi-k2-7-code-highspeed", Upstream: "moonshotai/Kimi-K2.7-Code-Highspeed", Provider: "command_code"},
	{ID: "cc-kimi-k3", Upstream: "moonshotai/Kimi-K3", Provider: "command_code"},

	// NVIDIA
	{ID: "cc-nemotron-3-ultra", Upstream: "nvidia/nemotron-3-ultra-550b-a55b", Provider: "command_code"},

	// OpenAI (GPT)
	{ID: "cc-gpt-5-3-codex", Upstream: "gpt-5.3-codex", Provider: "command_code"},
	{ID: "cc-gpt-5-4", Upstream: "gpt-5.4", Provider: "command_code"},
	{ID: "cc-gpt-5-4-mini", Upstream: "gpt-5.4-mini", Provider: "command_code"},
	{ID: "cc-gpt-5-5", Upstream: "gpt-5.5", Provider: "command_code"},
	{ID: "cc-gpt-5-6-luna", Upstream: "gpt-5.6-luna", Provider: "command_code"},
	{ID: "cc-gpt-5-6-sol", Upstream: "gpt-5.6-sol", Provider: "command_code"},
	{ID: "cc-gpt-5-6-terra", Upstream: "gpt-5.6-terra", Provider: "command_code"},

	// Poolside (Laguna)
	{ID: "cc-laguna-s-2-1-free", Upstream: "poolside/laguna-s-2.1-free", Provider: "command_code"},

	// Sakana AI (Fugu)
	{ID: "cc-fugu-ultra", Upstream: "sakana/fugu-ultra", Provider: "command_code"},

	// StepFun
	{ID: "cc-step-3-5-flash", Upstream: "stepfun/Step-3.5-Flash", Provider: "command_code"},
	{ID: "cc-step-3-7-flash", Upstream: "stepfun/Step-3.7-Flash", Provider: "command_code"},

	// Tencent
	{ID: "cc-tencent-hy3", Upstream: "tencent/hy3-paid", Provider: "command_code"},

	// Thinking Machines (Inkling)
	{ID: "cc-inkling", Upstream: "thinkingmachines/inkling", Provider: "command_code"},
	{ID: "cc-inkling-small", Upstream: "thinkingmachines/inkling-small", Provider: "command_code"},

	// xAI (Grok)
	{ID: "cc-grok-4-5", Upstream: "xai/grok-4.5", Provider: "command_code"},
	{ID: "cc-grok-4-6", Upstream: "xai/grok-4.6", Provider: "command_code"},

	// Xiaomi (MiMo)
	{ID: "cc-mimo-v2-5", Upstream: "xiaomi/mimo-v2.5", Provider: "command_code"},
	{ID: "cc-mimo-v2-5-pro", Upstream: "xiaomi/mimo-v2.5-pro", Provider: "command_code"},

	// Z AI (GLM)
	{ID: "cc-glm-5", Upstream: "zai-org/GLM-5", Provider: "command_code"},
	{ID: "cc-glm-5-1", Upstream: "zai-org/GLM-5.1", Provider: "command_code"},
	{ID: "cc-glm-5-2", Upstream: "zai-org/GLM-5.2", Provider: "command_code"},
	{ID: "cc-glm-5-2-fast", Upstream: "zai-org/GLM-5.2-Fast", Provider: "command_code"},
	{ID: "cc-glm-5-3", Upstream: "zai-org/GLM-5.3", Provider: "command_code"},
}
