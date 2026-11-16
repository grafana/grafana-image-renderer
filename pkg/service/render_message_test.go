package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRenderMessage(t *testing.T) {
	t.Parallel()

	t.Run("failure with an error is reported", func(t *testing.T) {
		t.Parallel()
		got := parseRenderMessage(`{"type":"REPORT_RENDER_COMPLETE","data":{"success":false,"error":{"status":404,"message":"Dashboard not found"}}}`)
		require.NotNil(t, got)
		require.Equal(t, 404, got.Status)
		require.Equal(t, "Dashboard not found", got.Message)
	})

	t.Run("failure without a status is still reported", func(t *testing.T) {
		t.Parallel()
		got := parseRenderMessage(`{"type":"REPORT_RENDER_COMPLETE","data":{"success":false,"error":{"message":"boom"}}}`)
		require.NotNil(t, got)
		require.Zero(t, got.Status)
		require.Equal(t, "boom", got.Message)
	})

	t.Run("long messages are truncated", func(t *testing.T) {
		t.Parallel()
		got := parseRenderMessage(`{"type":"REPORT_RENDER_COMPLETE","data":{"success":false,"error":{"message":"` + strings.Repeat("a", 5000) + `"}}}`)
		require.NotNil(t, got)
		require.Len(t, got.Message, maxRenderFailureMessageLen)
	})

	// Everything below must keep meaning "ready", as it did before the payload was read.
	for name, payload := range map[string]string{
		"success":                       `{"type":"REPORT_RENDER_COMPLETE","data":{"success":true}}`,
		"failure without an error":      `{"type":"REPORT_RENDER_COMPLETE","data":{"success":false}}`,
		"failure with an empty message": `{"type":"REPORT_RENDER_COMPLETE","data":{"success":false,"error":{"status":404,"message":"  "}}}`,
		"unknown type":                  `{"type":"SOMETHING_ELSE","data":{"success":false,"error":{"message":"x"}}}`,
		"not json":                      `done`,
		"empty":                         ``,
		"wrong shape":                   `{"type":"REPORT_RENDER_COMPLETE","data":"nope"}`,
	} {
		t.Run("treated as ready: "+name, func(t *testing.T) {
			t.Parallel()
			require.Nil(t, parseRenderMessage(payload))
		})
	}
}

func TestRenderFailedErrorIsMatchable(t *testing.T) {
	t.Parallel()

	var wrapped error = &RenderFailedError{Status: 403, Message: "no access"}
	var target *RenderFailedError
	require.True(t, errors.As(wrapped, &target))
	require.Contains(t, target.Error(), "403")
	require.Contains(t, target.Error(), "no access")
}
