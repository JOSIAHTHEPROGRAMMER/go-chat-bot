package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/logger"
)

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Chain wraps a handler with all middleware in order: Recovery, CORS, Logging.
func Chain(next http.HandlerFunc) http.HandlerFunc {
	return Recovery(CORS(Logging(next)))
}

// Logging attaches a RequestLog to the context, runs the handler,
// then prints a single structured log line when the request completes.
func Logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Attach a RequestLog so downstream layers can write into it
		ctx := logger.NewContext(r.Context())
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next(rw, r)

		// Build the log line from whatever the downstream layers filled in
		rl := logger.FromContext(ctx)
		extra := ""
		if rl != nil {
			extra = fmt.Sprintf(" plan=%s provider=%s docs=%d", rl.PlanType, rl.Provider, rl.DocCount)
		}

		log.Printf("%s %s %d %s%s", r.Method, r.URL.Path, rw.status, time.Since(start), extra)
	}
}

// CORS sets the headers needed for cross-origin requests from a browser.
// Set ALLOWED_ORIGIN in your .env, falls back to * for local development.
func CORS(next http.HandlerFunc) http.HandlerFunc {
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Preflight request, browsers send this before the real request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// Recovery catches any panic in a handler and returns a 500 instead of crashing the server.
func Recovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v\n%s", err, debug.Stack())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()

		next(w, r)
	}
}
