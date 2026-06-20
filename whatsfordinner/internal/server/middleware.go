package server

import (
	"net/http"
	"time"
)

// withMiddleware wraps a handler with the common middleware stack. The
// outermost middleware is applied last, so requests flow top-to-bottom.
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return s.logRequests(next)
}

// statusRecorder captures the response status code so it can be logged.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// logRequests logs one structured line per request, including method, path,
// status code and duration.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"remote", r.RemoteAddr,
			"duration", time.Since(start).String(),
		)
	})
}
