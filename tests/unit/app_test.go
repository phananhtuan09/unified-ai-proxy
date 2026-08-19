package unit

import (
	"testing"

	"github.com/tuanp-github/unified-ai-proxy/internal/app"
)

func TestBuildReturnsConfigErrorWithoutStartingServer(t *testing.T) {
	if _, err := app.Build(t.TempDir()+"/missing.yaml", nil); err == nil {
		t.Fatal("Build should fail for a missing config")
	}
}
