package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
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

// Flush forwards to the underlying ResponseWriter so SSE streaming works.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Chain wraps a handler with all middleware in order: Recovery, CORS, Auth, RateLimit, Logging.
func Chain(next http.HandlerFunc) http.HandlerFunc {
	return Recovery(CORS(Auth(RateLimit(Logging(next)))))
}

// Logging attaches a RequestLog to the context, runs the handler,
// then prints a single structured log line when the request completes.
func Logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx := logger.NewContext(r.Context())
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next(rw, r)

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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		// Preflight request, browsers send this before the real request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// Auth checks for a valid API key in the X-API-Key header.
// Set API_KEY in your .env. If unset, auth is skipped for local development.
func Auth(next http.HandlerFunc) http.HandlerFunc {
	apiKey := os.Getenv("API_KEY")

	return func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			next(w, r)
			return
		}

		if r.Header.Get("X-API-Key") != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// ipBucket tracks request timestamps for a single IP address.
type ipBucket struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// rateLimiter holds per-IP buckets and is safe for concurrent use.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	limit   int
	window  time.Duration
}

var limiter = &rateLimiter{
	buckets: make(map[string]*ipBucket),
	window:  time.Minute,
}

// RateLimit rejects requests from IPs that exceed RATE_LIMIT requests per minute.
// Defaults to 10 requests per minute if RATE_LIMIT is not set.
func RateLimit(next http.HandlerFunc) http.HandlerFunc {
	limit := 10
	if val := os.Getenv("RATE_LIMIT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			limit = n
		}
	}
	limiter.limit = limit

	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		limiter.mu.Lock()
		bucket, ok := limiter.buckets[ip]
		if !ok {
			bucket = &ipBucket{}
			limiter.buckets[ip] = bucket
		}
		limiter.mu.Unlock()

		bucket.mu.Lock()
		defer bucket.mu.Unlock()

		now := time.Now()
		cutoff := now.Add(-limiter.window)

		// Drop timestamps outside the current window
		valid := bucket.timestamps[:0]
		for _, t := range bucket.timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		bucket.timestamps = valid

		if len(bucket.timestamps) >= limiter.limit {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		bucket.timestamps = append(bucket.timestamps, now)
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
