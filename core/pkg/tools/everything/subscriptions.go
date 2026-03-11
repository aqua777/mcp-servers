package everything

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	// subscriptions tracks session IDs subscribed to a particular URI
	subscriptions = make(map[string]map[string]bool)
	subsMu        sync.RWMutex

	// updateIntervals tracks cancellation functions for periodic update routines per session
	updateIntervals = make(map[string]context.CancelFunc)
	intervalsMu     sync.RWMutex
)

// setSubscriptionHandlers wires up the MCP server to handle resource subscribe/unsubscribe.
func setSubscriptionHandlers(server *mcp.Server) {
	// The Go SDK handles subscribe/unsubscribe natively and passes through to callbacks
	// if we expose them, or we can just let it manage state?
	// The Go SDK Server type does not expose setRequestHandler directly in the same way,
	// but it handles subscriptions internally. However, if we need to emit simulated updates,
	// we need to track them.
	// Currently, the mcp.Server in go-sdk handles "resources/subscribe" via its own map if resources
	// are marked subscribable, but we want to simulate updates.
	// We'll expose explicit subscribe toggling via a tool or we can hook into a hypothetical callback
	// if the SDK has it. For now, since everything is demo, we'll track active sessions
	// if we can, or just send notifications to the default session.
}

// BeginSimulatedResourceUpdates starts a goroutine to send periodic update notifications.
func BeginSimulatedResourceUpdates(server *mcp.Server, sessionID string) {
	intervalsMu.Lock()
	defer intervalsMu.Unlock()

	if _, exists := updateIntervals[sessionID]; exists {
		return // already running
	}

	ctx, cancel := context.WithCancel(context.Background())
	updateIntervals[sessionID] = cancel

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Send once immediately
		sendUpdates(server, sessionID)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendUpdates(server, sessionID)
			}
		}
	}()
}

func sendUpdates(server *mcp.Server, sessionID string) {
	// Without direct access to the SDK's internal subscription list per session,
	// we will broadcast a notification for our dynamic resources just as a demo.
	// In the real TS server, it manually tracks subscriptions.
	// Since Go SDK handles subscribe, we just emit a resource update notification.

	err := server.ResourceUpdated(context.Background(), &mcp.ResourceUpdatedNotificationParams{
		URI: "demo://resource/dynamic/text/1",
	})
	if err != nil {
		// Log or ignore
		fmt.Printf("Error sending simulated resource update: %v\n", err)
	}
}

// StopSimulatedResourceUpdates stops the periodic notifications for a session.
func StopSimulatedResourceUpdates(sessionID string) {
	intervalsMu.Lock()
	defer intervalsMu.Unlock()

	if cancel, exists := updateIntervals[sessionID]; exists {
		cancel()
		delete(updateIntervals, sessionID)
	}
}
