package main

import (
	"fmt"
	"github.com/justinas/nosurf"
	"net/http"
	"server_monitor/internal/helpers"
	"strconv"
	"strings"
	"time"
)

// SessionLoad loads the session on requests
func SessionLoad(next http.Handler) http.Handler {
	return session.LoadAndSave(next)
}

// Auth checks for authentication
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !helpers.IsAuthenticated(r) {
			url := r.URL.Path
			http.Redirect(w, r, fmt.Sprintf("/?target=%s", url), http.StatusFound)
			return
		}
		w.Header().Add("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// RecoverPanic recovers from a panic
func RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				helpers.ServerError(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// NoSurf implements CSRF protection
func NoSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.ExemptPath("/pusher/auth")
	csrfHandler.ExemptPath("/pusher/hook")

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   app.InProduction,
		SameSite: http.SameSiteStrictMode,
		Domain:   app.Domain,
	})

	return csrfHandler
}

// CheckRemember checks to see if we should log the user in automatically
func CheckRemember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(fmt.Sprintf("_%s_gowatcher_remember", preferenceMap["identifier"]))
		if err == nil {
			key := cookie.Value
			if len(key) > 0 {
				id, hash := getIdAndHashFromKey(key)
				isValidHash := repo.DB.CheckForToken(id, hash)
				isAuthenticated := helpers.IsAuthenticated(r)
				if !isAuthenticated && isValidHash {
					// valid remember me token, so log the user in
					putSessionDataWithValidToken(r, id)
				} else if !isValidHash {
					// invalid token, so delete the cookie
					deleteRememberCookie(w, r)
					putSessionDataWithInvalidToken(r)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// getIdAndHashFromKey extracts user_id and token from key
func getIdAndHashFromKey(key string) (int, string) {
	split := strings.Split(key, "|")
	uid, hash := split[0], split[1]
	id, _ := strconv.Atoi(uid)
	return id, hash
}

// putSessionDataWithValidToken puts key-value data to the session
func putSessionDataWithValidToken(r *http.Request, id int) {
	_ = session.RenewToken(r.Context())
	user, _ := repo.DB.GetUserById(id)
	hashedPassword := user.Password
	session.Put(r.Context(), "userID", id)
	session.Put(r.Context(), "userName", user.Name)
	session.Put(r.Context(), "hashedPassword", string(hashedPassword))
	session.Put(r.Context(), "user", user)
}

// putSessionDataWithInvalidToken puts key(error)-value data to the session
func putSessionDataWithInvalidToken(r *http.Request) {
	session.Put(r.Context(), "error", "You've been logged out from another device!")
}

// deleteRememberCookie deletes the remember me cookie, and logs the user out
func deleteRememberCookie(w http.ResponseWriter, r *http.Request) {
	_ = session.RenewToken(r.Context())

	// delete the cookie
	newCookie := http.Cookie{
		Name:     fmt.Sprintf("_%s_ggowatcher_remember", preferenceMap["identifier"]),
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-100 * time.Hour),
		HttpOnly: true,
		Domain:   app.Domain,
		MaxAge:   -1,
		Secure:   app.InProduction,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &newCookie)

	// log them out
	session.Remove(r.Context(), "userID")
	_ = session.Destroy(r.Context())
	_ = session.RenewToken(r.Context())
}
