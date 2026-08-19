package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tuanp-github/unified-ai-proxy/internal/config"
	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// transport owns shared provider metadata and bounded HTTP helpers.
// Authentication lifecycle belongs to the provider capability that needs it.
type transport struct {
	name   string
	cfg    config.ProviderConfig
	models []model.Model
	client *http.Client
}

func newTransport(name string, cfg config.ProviderConfig, models []model.Model, timeout time.Duration) transport {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return transport{
		name:   name,
		cfg:    cfg,
		models: models,
		client: &http.Client{Timeout: timeout},
	}
}

func (t *transport) Name() string          { return t.name }
func (t *transport) Models() []model.Model { return t.models }

// doJSON performs a bounded JSON request and returns the raw status and body.
func (t *transport) doJSON(ctx context.Context, method, endpoint string, headers map[string]string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}

// readBody drains an error response body up to a bounded size.
func readBody(resp *http.Response) []byte {
	if resp.Body == nil {
		return nil
	}
	return func() []byte {
		returnBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return returnBytes
	}()
}

func isRetryStatus(status int) bool {
	switch status {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func isAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}
