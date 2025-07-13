package churro

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Middleware defines the middleware function type
type Middleware func(next http.Handler) http.Handler

// Recovery middleware recovers from panics and returns a structured error response
func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic
					slog.Error("Panic recovered",
						"error", err,
						"method", r.Method,
						"url", r.URL.String(),
						"stack", string(debug.Stack()),
					)

					// Return a structured error response
					problemDetails := NewInternalServerError("An unexpected error occurred")
					problemDetails.WithExtension("panic", fmt.Sprintf("%v", err))

					if !WriteProblemDetails(w, problemDetails) {
						// Fallback if problem details writing fails
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware adds Cross-Origin Resource Sharing headers
func CORS(options CORSOptions) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			if options.AllowOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", options.AllowOrigin)
			}
			if options.AllowMethods != "" {
				w.Header().Set("Access-Control-Allow-Methods", options.AllowMethods)
			}
			if options.AllowHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", options.AllowHeaders)
			}
			if options.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if options.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", options.MaxAge))
			}

			// Handle preflight request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSOptions configures the CORS middleware
type CORSOptions struct {
	AllowOrigin      string
	AllowMethods     string
	AllowHeaders     string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSOptions returns sensible CORS defaults
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowOrigin:      "*",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// RequestID middleware adds a unique request ID to each request
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := generateRequestID()
			w.Header().Set("X-Request-ID", requestID)

			// Add to context for use in handlers
			ctx := WithRequestID(r.Context(), requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Security middleware adds common security headers
func Security() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			next.ServeHTTP(w, r)
		})
	}
}

// Timeout middleware wraps the handler with a timeout
func Timeout(timeout int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next,
			timeoutDuration(timeout),
			"Request timeout")
	}
}

// Logger middleware logs HTTP requests
func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

			// Get request ID if available
			requestID := GetRequestID(r.Context())

			slog.Info("HTTP Request",
				"method", r.Method,
				"url", r.URL.String(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"request_id", requestID,
			)

			next.ServeHTTP(wrapped, r)

			slog.Info("HTTP Response",
				"method", r.Method,
				"url", r.URL.String(),
				"status", wrapped.statusCode,
				"request_id", requestID,
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
