package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"server_monitor/internal/helpers"
	"server_monitor/internal/models"
	"strings"
	"time"
)

// LoginScreen shows the home (login) screen
func (repo *DBRepo) LoginScreen(w http.ResponseWriter, r *http.Request) {
	// if already logged in, take to dashboard
	if repo.App.Session.Exists(r.Context(), "userID") {
		http.Redirect(w, r, "/admin/overview", http.StatusSeeOther)
		return
	}

	err := helpers.RenderPage(w, r, "login", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// Login attempts to log the user in
func (repo *DBRepo) Login(w http.ResponseWriter, r *http.Request) {
	_ = repo.App.Session.RenewToken(r.Context())
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		ClientError(w, r, http.StatusBadRequest)
		return
	}
	id, hash, err := repo.DB.Authenticate(r.Form.Get("email"), r.Form.Get("password"))
	if err == models.ErrInvalidCredentials {
		app.Session.Put(r.Context(), "error", "Invalid login")
		err := helpers.RenderPage(w, r, "login", nil, nil)
		if err != nil {
			printTemplateError(w, err)
		}
		return
	} else if err != nil {
		log.Println(err)
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	if r.Form.Get("remember") == "remember" {
		randomString := helpers.RandomString(12)
		hasher := sha256.New()

		_, err = hasher.Write([]byte(randomString))
		if err != nil {
			log.Println(err)
		}

		sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		err = repo.DB.InsertRememberMeToken(id, sha)
		if err != nil {
			log.Println(err)
		}

		// write a cookie
		expire := time.Now().Add(365 * 24 * 60 * 60 * time.Second)
		cookie := http.Cookie{
			Name:     fmt.Sprintf("_%s_gowatcher_remember", app.PreferenceMap["identifier"]),
			Value:    fmt.Sprintf("%d|%s", id, sha),
			Path:     "/",
			Expires:  expire,
			HttpOnly: true,
			Domain:   app.Domain,
			MaxAge:   365 * 24 * 60 * 60,
			Secure:   app.InProduction,
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(w, &cookie)
	}

	user, err := repo.DB.GetUserById(id)
	if err != nil {
		log.Println(err)
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	app.Session.Put(r.Context(), "userID", id)
	app.Session.Put(r.Context(), "hashedPassword", hash)
	app.Session.Put(r.Context(), "flash", "You've been logged in successfully!")
	app.Session.Put(r.Context(), "user", user)

	if r.Form.Get("target") != "" {
		http.Redirect(w, r, r.Form.Get("target"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/overview", http.StatusSeeOther)
}

// Logout logs the user out
func (repo *DBRepo) Logout(w http.ResponseWriter, r *http.Request) {
	// delete the remember_me_token, if any
	cookie, err := r.Cookie(fmt.Sprintf("_%s_gowatcher_remember", app.PreferenceMap["identifier"]))
	if err == nil {
		key := cookie.Value
		if len(key) > 0 {
			split := strings.Split(key, "|")
			token := split[1]
			err = repo.DB.DeleteToken(token)
			if err != nil {
				log.Println(err)
			}
		}
	}

	delCookie := http.Cookie{
		Name:     fmt.Sprintf("_%s_gowatcher_remember", app.PreferenceMap["identifier"]),
		Value:    "",
		Domain:   app.Domain,
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
	}
	http.SetCookie(w, &delCookie)
	_ = app.Session.RenewToken(r.Context())
	_ = app.Session.Destroy(r.Context())
	_ = app.Session.RenewToken(r.Context())

	repo.App.Session.Put(r.Context(), "flash", "You've been logged out successfully!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
