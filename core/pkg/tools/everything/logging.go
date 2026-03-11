package everything

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	loggingIntervals = make(map[string]context.CancelFunc)
	loggingMu        sync.RWMutex
)

// BeginSimulatedLogging starts a goroutine to send periodic log messages.
func BeginSimulatedLogging(server *mcp.Server, sessionID string) {
	loggingMu.Lock()
	defer loggingMu.Unlock()

	if _, exists := loggingIntervals[sessionID]; exists {
		return // already running
	}

	ctx, cancel := context.WithCancel(context.Background())
	loggingIntervals[sessionID] = cancel

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Send once immediately
		sendLog(server, sessionID)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendLog(server, sessionID)
			}
		}
	}()
}

func sendLog(server *mcp.Server, sessionID string) {
	levels := []mcp.LoggingLevel{
		"debug",
		"info",
		"notice",
		"warning",
		"error",
		"critical",
		"alert",
		"emergency",
	}

	level := levels[rand.Intn(len(levels))]
	data := string(level) + "-level message"
	if sessionID != "" {
		data += " - SessionId " + sessionID
	}

	params := &mcp.LoggingMessageParams{
		Level: level,
		Data:  data,
	}

	for session := range server.Sessions() {
		if sessionID == "" || session.ID() == sessionID {
			_ = session.Log(context.Background(), params)
		}
	}
}

// StopSimulatedLogging stops the periodic log messages for a session.
func StopSimulatedLogging(sessionID string) {
	loggingMu.Lock()
	defer loggingMu.Unlock()

	if cancel, exists := loggingIntervals[sessionID]; exists {
		cancel()
		delete(loggingIntervals, sessionID)
	}
}
