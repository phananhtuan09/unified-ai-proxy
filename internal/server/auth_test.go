package server

import "testing"

func TestLocalAPIKey(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		anthropicKey  string
		want          string
	}{
		{name: "bearer", authorization: "Bearer bearer-key", anthropicKey: "anthropic-key", want: "bearer-key"},
		{name: "anthropic", anthropicKey: "anthropic-key", want: "anthropic-key"},
		{name: "trim anthropic key", anthropicKey: "  anthropic-key  ", want: "anthropic-key"},
		{name: "missing", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := localAPIKey(tt.authorization, tt.anthropicKey); got != tt.want {
				t.Fatalf("localAPIKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
