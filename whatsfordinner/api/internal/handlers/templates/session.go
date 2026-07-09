package handlers

import (
	"net/http"
	"strconv"
	"time"
)

// sessionCookieName is the cookie that stores which user is "signed in".
//
// There is intentionally no password: this app runs on a shared household
// device, so "signing in" just means picking who you are from the users
// list at /login. The cookie is a convenience, not a security boundary.
const sessionCookieName = "wfd_user_id"

// setSessionUser marks the given user as the active one on this browser.
func setSessionUser(w http.ResponseWriter, userID int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    strconv.FormatInt(userID, 10),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().AddDate(1, 0, 0),
	})
}

// clearSessionUser signs the current browser out.
func clearSessionUser(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// sessionUserID reads the signed-in user's ID from the request, if any.
func sessionUserID(r *http.Request) (int64, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return 0, false
	}

	id, err := strconv.ParseInt(c.Value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
