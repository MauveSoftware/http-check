package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPClientUsesDialContext(t *testing.T) {
	s := &HTTPCheckServer{reqTimeout: time.Second, tlsTimeout: time.Second}

	tr, ok := s.newHTTPClient(false).Transport.(*http.Transport)
	require.True(t, ok, "transport should be *http.Transport")

	assert.NotNil(t, tr.DialContext, "DialContext should be set so dials can be cancelled")
}
