package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/logs"
	"github.com/tuanp-github/unified-ai-proxy/internal/proxy"
	"github.com/tuanp-github/unified-ai-proxy/internal/version"
)

// Server wires the HTTP routes to the routing service.
type Server struct {
	cfg     *config.Config
	svc     *proxy.Service
	engine  *gin.Engine
	apiKeys map[string]bool
	logger  *logs.Store
}

// New builds the HTTP server. A nil logger disables request logging.
func New(cfg *config.Config, svc *proxy.Service, logger *logs.Store) *Server {
	s := &Server{
		cfg:     cfg,
		svc:     svc,
		engine:  gin.New(),
		apiKeys: make(map[string]bool, len(cfg.Server.APIKeys)),
		logger:  logger,
	}
	for _, k := range cfg.Server.APIKeys {
		s.apiKeys[k] = true
	}
	s.routes()
	return s
}

// Handler exposes the underlying http.Handler.
func (s *Server) Handler() http.Handler { return s.engine }

// Addr returns the configured listen address.
func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
}

func (s *Server) routes() {
	s.engine.Use(gin.Recovery())
	s.engine.Use(s.requestLogger())

	s.engine.GET("/health", s.handleHealth)

	v1 := s.engine.Group("/v1", s.authMiddleware())
	{
		v1.GET("/models", s.handleModels)
		v1.POST("/chat/completions", s.handleChatCompletions)
		v1.POST("/messages", s.handleMessages)
	}
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			abortAuth(c)
			return
		}
		key := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		if !s.apiKeys[key] {
			abortAuth(c)
			return
		}
		c.Next()
	}
}

func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.logger == nil {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		s.logger.Add(logs.Entry{
			Time:    start,
			Method:  c.Request.Method,
			Path:    c.Request.URL.Path,
			Status:  c.Writer.Status(),
			Latency: time.Since(start),
		})
	}
}

func abortAuth(c *gin.Context) {
	e := apierr.Unauthorized("missing or invalid API key")
	if strings.HasSuffix(c.FullPath(), "/messages") {
		writeAnthropicError(c, e)
		return
	}
	writeOpenAIError(c, e)
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": version.Version,
	})
}

func (s *Server) handleModels(c *gin.Context) {
	models := s.svc.Models()
	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		data = append(data, gin.H{
			"id":       m.ID,
			"object":   "model",
			"owned_by": ownedBy(m.Provider),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func ownedBy(provider string) string {
	switch provider {
	case "openai_codex":
		return "openai-codex"
	default:
		return provider
	}
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:              s.Addr(),
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}
