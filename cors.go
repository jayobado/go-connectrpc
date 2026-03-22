package connectrpc

import (
	"net/http"
	"strconv"
	"strings"
)

type CorsConfig struct {
	AllowedOrigins []string
	AllowedHeaders []string
	MaxAge         int
}

func defaultCORSConfig(origins []string) CorsConfig {
	return CorsConfig{
		AllowedOrigins: origins,
		AllowedHeaders: []string{
			"Content-Type",
			"Connect-Protocol-Version",
			"Connect-Timeout-Ms",
			"Authorization",
			"Cookie",
			"X-Request-Id",
		},
		MaxAge: 7200,
	}
}

func isAllowedOrigin(allowed []string, origin string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

func corsMiddleware(cfg CorsConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !isAllowedOrigin(cfg.AllowedOrigins, origin) {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers",
				"Content-Type, Connect-Protocol-Version, Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
			w.Header().Set("Vary", "Origin")

			if r.Method == http.MethodOptions {
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}