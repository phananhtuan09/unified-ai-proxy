package server

import (
	"crypto/rand"
	"fmt"

	"github.com/tuanp-github/unified-ai-proxy/internal/apierr"
)

// asAPIError normalizes any error into an *apierr.APIError.
func asAPIError(err error) *apierr.APIError {
	if err == nil {
		return nil
	}
	if ae, ok := err.(*apierr.APIError); ok {
		return ae
	}
	return apierr.ProviderUnavailable(err.Error())
}

// randomID returns a random lowercase hex identifier.
func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", b)
	}
	return fmt.Sprintf("%x", b)
}
