package server

import (
	"github.com/gin-gonic/gin"

	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
)

// writeOpenAIError writes an OpenAI-compatible error body.
func writeOpenAIError(c *gin.Context, e *apierr.APIError) {
	c.AbortWithStatusJSON(e.Status, gin.H{
		"error": gin.H{
			"message": e.Message,
			"type":    e.ErrorType(),
			"code":    e.Code,
		},
	})
}

// writeAnthropicError writes an Anthropic-compatible error body.
func writeAnthropicError(c *gin.Context, e *apierr.APIError) {
	c.AbortWithStatusJSON(e.Status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    e.ErrorType(),
			"message": e.Message,
		},
	})
}
