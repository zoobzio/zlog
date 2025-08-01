package zlog

import (
	"context"
	"runtime"
	"sync"
)

// contextStore manages goroutine-local context storage for ephemeral context creation.
// This allows zlog to propagate context (for distributed tracing, etc.) without
// requiring users to pass context to every Emit() call.
type contextStore struct {
	contexts map[int64]context.Context
	mu       sync.RWMutex
}

var store = &contextStore{
	contexts: make(map[int64]context.Context),
}

// getGoroutineID returns the current goroutine ID.
// This is used as a key for goroutine-local context storage.
func getGoroutineID() int64 {
	// This is a hack to get the goroutine ID from the stack trace
	// In production code, you might want to use a library like
	// github.com/petermattis/goid for better performance
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	// Parse "goroutine 123 [running]:"
	// Find the space after "goroutine"
	start := 10
	for i := start; i < n; i++ {
		if buf[i] == ' ' {
			// Found the end of the goroutine ID
			var id int64
			for j := start; j < i; j++ {
				id = id*10 + int64(buf[j]-'0')
			}
			return id
		}
	}
	return 0
}

// SetContext stores a context for the current goroutine.
// This context will be used for subsequent Emit() calls within this goroutine
// until ClearContext() is called or the goroutine ends.
//
// This enables context propagation without changing the Emit() API:
//
//	func handler(ctx context.Context, req *http.Request) {
//	    zlog.SetContext(ctx)  // Set once
//	    defer zlog.ClearContext()
//
//	    // All logging calls in this goroutine will use the context
//	    zlog.Info("Processing request", zlog.String("path", req.URL.Path))
//
//	    processPayment() // This function's logs will also use the context
//	}
//
// Middleware can automatically set context for HTTP handlers:
//
//	func LoggingMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        // Extract trace context from headers
//	        ctx := trace.FromHTTPHeaders(r.Context(), r.Header)
//	        zlog.SetContext(ctx)
//	        defer zlog.ClearContext()
//
//	        next.ServeHTTP(w, r)
//	    })
//	}
func SetContext(ctx context.Context) {
	if ctx == nil {
		return
	}

	gid := getGoroutineID()
	store.mu.Lock()
	store.contexts[gid] = ctx
	store.mu.Unlock()
}

// ClearContext removes the stored context for the current goroutine.
// This should typically be called with defer to ensure cleanup.
func ClearContext() {
	gid := getGoroutineID()
	store.mu.Lock()
	delete(store.contexts, gid)
	store.mu.Unlock()
}

// getContext retrieves the context for the current goroutine.
// Returns context.Background() if no context has been set.
func getContext() context.Context {
	gid := getGoroutineID()
	store.mu.RLock()
	ctx, exists := store.contexts[gid]
	store.mu.RUnlock()

	if !exists {
		return context.Background()
	}
	return ctx
}

// WithContext returns a function that temporarily sets a context for the current goroutine.
// The returned function should be called to restore the previous context.
//
// This is useful for scoped context usage:
//
//	restore := zlog.WithContext(traceCtx)
//	defer restore()
//
//	zlog.Info("This will use traceCtx")
func WithContext(ctx context.Context) func() {
	gid := getGoroutineID()

	// Save the current context (if any)
	store.mu.RLock()
	oldCtx, hadOldCtx := store.contexts[gid]
	store.mu.RUnlock()

	// Set the new context
	SetContext(ctx)

	// Return a function to restore the old state
	return func() {
		store.mu.Lock()
		if hadOldCtx {
			store.contexts[gid] = oldCtx
		} else {
			delete(store.contexts, gid)
		}
		store.mu.Unlock()
	}
}
