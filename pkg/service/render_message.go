package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

// maxRenderFailureMessageLen bounds how much of the page-supplied message is echoed back to the caller.
const maxRenderFailureMessageLen = 1024

// RenderFailedError is returned when the page tells the renderer, over the message channel, that it
// gave up loading and that capturing it would not produce anything useful (for example, the
// dashboard does not exist or the user may not view it).
type RenderFailedError struct {
	// Status is the HTTP-like status the page associated with the failure, or 0 if it gave none.
	Status int
	// Message is the human-readable reason supplied by the page.
	Message string
}

func (e *RenderFailedError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("page reported a failure (status %d): %s", e.Status, e.Message)
	}
	return "page reported a failure: " + e.Message
}

type renderCompleteMessage struct {
	Type string `json:"type"`
	Data struct {
		Success bool `json:"success"`
		Error   *struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"data"`
}

// parseRenderMessage interprets a payload sent to __grafanaImageRendererMessageChannel.
//
// It returns a *RenderFailedError only for a REPORT_RENDER_COMPLETE message with success=false that
// carries an error with a message. Everything else, including payloads that are not JSON, unknown
// message types, and success=false without an error, is treated as "ready", which is what the
// channel meant before messages had a payload that was read. That keeps older and newer Grafana
// versions, and other callers such as notebooks that report success=false for a partial capture,
// working unchanged.
func parseRenderMessage(payload string) *RenderFailedError {
	var msg renderCompleteMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return nil
	}
	if msg.Type != "REPORT_RENDER_COMPLETE" || msg.Data.Success || msg.Data.Error == nil {
		return nil
	}
	reason := strings.TrimSpace(msg.Data.Error.Message)
	if reason == "" {
		return nil
	}
	if len(reason) > maxRenderFailureMessageLen {
		reason = strings.ToValidUTF8(reason[:maxRenderFailureMessageLen], "")
	}
	return &RenderFailedError{Status: msg.Data.Error.Status, Message: reason}
}
