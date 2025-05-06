package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Define a context key type for the logger.
type contextKey string

// LoggerKey is the key used to store the request-scoped logger in the context.
// Exported for use in other packages if direct context access is needed, though LoggerFromContext is preferred.
const LoggerKey = contextKey("logger")

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	// Default to 200 OK if WriteHeader not called
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs HTTP request and response details using slog
// and injects a request-scoped logger into the context.
func LoggingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		tstart := time.Now()
		rw := newResponseWriter(w)

		// Create request-scoped logger with initial attributes
		requestID := uuid.New().String()
		logger := slog.Default().With(
			slog.String("request_id", requestID),
			slog.String("http.request.method", r.Method),
			slog.String("http.url.path", r.URL.Path),
			slog.String("user_agent.original", r.UserAgent()),
			slog.String("client.address", r.RemoteAddr), // May be empty or inaccurate
		)

		// Add logger to context
		ctx := context.WithValue(r.Context(), LoggerKey, logger)

		// Serve the request with the new context
		next.ServeHTTP(rw, r.WithContext(ctx))

		duration := time.Since(tstart)

		// Extract UID from context if available (set by auth middleware)
		// Use the *original* context 'r.Context()' here if auth middleware runs *after* logging middleware's context injection
		// If auth runs *before*, then 'ctx' already has the UID and logger. Let's assume auth runs before for simplicity.
		uid, _ := GetDeviceIDFromContext(ctx) // Check the enhanced context

		// Log final request details using the request-scoped logger from context
		requestLogger := logger.With(slog.String("user.id", uid)) // Add uid if found
		requestLogger.InfoContext(ctx, "HTTP request processed",  // Pass context
			slog.Int("http.response.status_code", rw.statusCode),
			slog.Duration("duration", duration),
		)
	}
	return http.HandlerFunc(fn)
}

// LoggerFromContext retrieves the logger from the context.
// If no logger is found, it returns the default logger.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
