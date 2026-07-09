package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionUserRoundTrip(t *testing.T) {
	rec := httptest.NewRecorder()
	setSessionUser(rec, 42)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	id, ok := sessionUserID(req)
	if !ok || id != 42 {
		t.Fatalf("expected session user id 42, got %d (ok=%v)", id, ok)
	}
}

func TestSessionUserAbsentByDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if _, ok := sessionUserID(req); ok {
		t.Fatal("expected no session user on a request without a cookie")
	}
}

func TestClearSessionUser(t *testing.T) {
	rec := httptest.NewRecorder()
	clearSessionUser(rec)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one Set-Cookie header, got %d", len(cookies))
	}
	if cookies[0].MaxAge >= 0 {
		t.Fatalf("expected clearSessionUser to expire the cookie, got MaxAge=%d", cookies[0].MaxAge)
	}
}
