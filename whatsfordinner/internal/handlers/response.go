package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// maxBodyBytes caps the size of a JSON request body (1 MiB).
const maxBodyBytes = 1 << 20

// writeJSON sets the JSON content type and writes payload with the given status
// code. A nil payload writes only the status (e.g. for 204 No Content).
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

// writeError writes a JSON error envelope: {"error": "..."}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// decodeJSON decodes a single JSON object from the request body into dst,
// rejecting unknown fields and oversized or trailing content.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

// respondStoreError maps a store error to an appropriate HTTP response.
func (h *Handler) respondStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "resource already exists")
	case errors.Is(err, store.ErrReference):
		writeError(w, http.StatusUnprocessableEntity, "referenced resource does not exist or is still in use")
	default:
		h.logger.Error("store error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
