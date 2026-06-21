package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
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

// requireAPIKey enforces that every request carries a valid UUID in the
// X-Api-Key header. The UUID is looked up in the api_keys table; a missing,
// malformed, or unknown key results in 401 Unauthorized.
func (s *Server) requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-Api-Key")
		if raw == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing API key")
			return
		}

		key, err := uuid.Parse(raw)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		ok, err := s.store.APIKeyExists(r.Context(), key)
		if err != nil {
			s.logger.Error("API key lookup failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSONError writes a {"error":"…"} JSON body with the given status code.
// Defined here so the server package does not need to import the handlers package
// just to write an auth error.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
