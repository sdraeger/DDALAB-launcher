package interrupt

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Handler manages interrupt signals for graceful cancellation
type Handler struct {
	mu       sync.RWMutex
	active   bool
	cancel   context.CancelFunc
	notifyCh chan struct{}
}

// NewHandler creates a new interrupt handler
func NewHandler() *Handler {
	return &Handler{
		notifyCh: make(chan struct{}, 1),
	}
}

// WithCancellableContext creates a context that can be cancelled by interrupt signals
func (h *Handler) WithCancellableContext(parent context.Context) (context.Context, context.CancelFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ctx, cancel := context.WithCancel(parent)
	h.cancel = cancel
	h.active = true

	// Set up signal handling
	go h.handleSignals()

	return ctx, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.active = false
		if h.cancel != nil {
			h.cancel()
		}
	}
}

// handleSignals listens for interrupt signals and cancels the context
func (h *Handler) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		h.mu.RLock()
		active := h.active
		cancel := h.cancel
		h.mu.RUnlock()

		if active && cancel != nil {
			fmt.Println("\n⚠️  Operation interrupted by user")
			cancel()

			// Notify that interruption occurred
			select {
			case h.notifyCh <- struct{}{}:
			default:
			}
		}
	}

	signal.Stop(sigCh)
}

// WasInterrupted checks if the last operation was interrupted
func (h *Handler) WasInterrupted() bool {
	// Non-blocking receive to check if there's a notification
	// Using select with default is intentional for non-blocking behavior
	select {
	case <-h.notifyCh:
		return true
	default:
		return false
	}
}

// IsInterruptError checks if an error is due to context cancellation
func IsInterruptError(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}
