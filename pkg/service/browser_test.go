package service

import (
	"testing"

	"github.com/chromedp/cdproto/network"
	"github.com/stretchr/testify/assert"
)

func TestShouldTrackReadinessNetworkRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		event    *network.EventRequestWillBeSent
		expected bool
	}{
		{
			name:     "nil event",
			event:    nil,
			expected: false,
		},
		{
			name:     "nil request payload",
			event:    &network.EventRequestWillBeSent{},
			expected: false,
		},
		{
			name: "http URL",
			event: &network.EventRequestWillBeSent{
				Request: &network.Request{URL: "http://example.com/path"},
			},
			expected: true,
		},
		{
			name: "https URL uppercase scheme",
			event: &network.EventRequestWillBeSent{
				Request: &network.Request{URL: "HTTPS://example.com/path"},
			},
			expected: true,
		},
		{
			name: "blob URL",
			event: &network.EventRequestWillBeSent{
				Request: &network.Request{URL: "blob:http://example.com/worker"},
			},
			expected: false,
		},
		{
			name: "event source request",
			event: &network.EventRequestWillBeSent{
				Type:    network.ResourceTypeEventSource,
				Request: &network.Request{URL: "https://example.com/events"},
			},
			expected: false,
		},
		{
			name: "websocket request",
			event: &network.EventRequestWillBeSent{
				Type:    network.ResourceTypeWebSocket,
				Request: &network.Request{URL: "wss://example.com/live"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, shouldTrackReadinessNetworkRequest(tt.event))
		})
	}
}

func TestReadinessNetworkObserverTracksUniqueRequestIDs(t *testing.T) {
	t.Parallel()

	observer := newReadinessNetworkObserver()

	first := &network.EventRequestWillBeSent{
		RequestID: network.RequestID("req-1"),
		Request:   &network.Request{URL: "https://example.com/main.js"},
	}
	assert.True(t, observer.onRequestWillBeSent(first))
	assert.Equal(t, int64(1), observer.inflight())

	// Redirect / duplicate request events for the same RequestID should not increase inflight count.
	assert.False(t, observer.onRequestWillBeSent(first))
	assert.Equal(t, int64(1), observer.inflight())

	observer.onRequestCompleted(network.RequestID("req-1"))
	assert.Equal(t, int64(0), observer.inflight())

	// Duplicate completion events should be ignored and never make inflight negative.
	observer.onRequestCompleted(network.RequestID("req-1"))
	assert.Equal(t, int64(0), observer.inflight())
}

func TestReadinessNetworkObserverIgnoresUntrackedRequests(t *testing.T) {
	t.Parallel()

	observer := newReadinessNetworkObserver()

	assert.False(t, observer.onRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: network.RequestID("blob-1"),
		Request:   &network.Request{URL: "blob:https://example.com/worker"},
	}))
	assert.Equal(t, int64(0), observer.inflight())

	observer.onRequestCompleted(network.RequestID("blob-1"))
	assert.Equal(t, int64(0), observer.inflight())
}
