package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler.ServeHTTP(w, r)
		slog.Info(
			"request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.Path),
			slog.Duration("time", time.Since(start)),
		)
	})
}
