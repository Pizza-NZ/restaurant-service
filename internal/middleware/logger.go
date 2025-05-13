package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logger is a middleware that logs HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture the status code
		lw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(lw, r)

		// Log the request
		duration := time.Since(start)
		log.Printf("%s %s %s %d %s", r.RemoteAddr, r.Method, r.URL.Path, lw.statusCode, duration)
	})
}

// loggingResponseWriter is a wrapper around http.ResponseWriter to capture the status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before calling the underlying ResponseWriter
func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}
