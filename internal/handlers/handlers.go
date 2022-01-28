package handlers

import (
	"fmt"
	"github.com/CloudyKit/jet/v6"
	"github.com/go-chi/chi"
	"log"
	"net/http"
	"runtime/debug"
	"server_monitor/internal/config"
	"server_monitor/internal/driver"
	"server_monitor/internal/helpers"
	"server_monitor/internal/models"
	"server_monitor/internal/repository"
	"server_monitor/internal/repository/dbrepo"
	"strconv"
)

var Repo *DBRepo
var app *config.AppConfig

type DBRepo struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

func NewHandlers(repo *DBRepo, a *config.AppConfig) {
	Repo = repo
	app = a
}

func NewMysqlHandlers(db *driver.DB, a *config.AppConfig) *DBRepo {
	return &DBRepo{
		App: a,
		DB:  dbrepo.NewMysqlRepo(db.SQL, a),
	}
}

// AdminDashboard displays the dashboard
func (repo *DBRepo) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	vars := make(jet.VarMap)
	vars.Set("no_healthy", 0)
	vars.Set("no_problem", 0)
	vars.Set("no_pending", 0)
	vars.Set("no_warning", 0)

	err := helpers.RenderPage(w, r, "dashboard", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// Events displays the events page
func (repo *DBRepo) Events(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "events", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// Settings display the settings page
func (repo *DBRepo) Settings(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "settings", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// PostSettings saves site settings
func (repo *DBRepo) PostSettings(w http.ResponseWriter, r *http.Request) {
	prefMap := getPreferenceMapData(r)

	err := repo.DB.InsertOrUpdateSitePreferences(prefMap)
	if err != nil {
		log.Println(err)
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	// update app config
	for k, v := range prefMap {
		app.PreferenceMap[k] = v
	}

	app.Session.Put(r.Context(), "flash", "Changes saved")

	if r.Form.Get("action") == "1" {
		http.Redirect(w, r, "/admin/overview", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
	}
}

// AllHosts displays list of all hosts
func (repo *DBRepo) AllHosts(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "hosts", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// Host shows the host add/edit form
func (repo *DBRepo) Host(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "host", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// AllUsers lists all admin users
func (repo *DBRepo) AllUsers(w http.ResponseWriter, r *http.Request) {
	vars := make(jet.VarMap)

	u, err := repo.DB.AllUsers()
	if err != nil {
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	vars.Set("users", u)
	err = helpers.RenderPage(w, r, "users", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// OneUser displays the add/edit user page
func (repo *DBRepo) OneUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Println(err)
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	vars := make(jet.VarMap)

	if id > 0 {
		user, err := repo.DB.GetUserById(id)
		if err != nil {
			ClientError(w, r, http.StatusBadRequest)
			return
		}

		vars.Set("user", user)
	} else {
		var u models.User
		vars.Set("user", u)
	}

	err = helpers.RenderPage(w, r, "user", vars, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

// PostOneUser adds/edits a user
func (repo *DBRepo) PostOneUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		ClientError(w, r, http.StatusBadRequest)
		return
	}

	var user models.User

	if id > 0 {
		user, _ = repo.DB.GetUserById(id)
		user.Name = r.Form.Get("name")
		user.Email = r.Form.Get("email")
		user.UserActive, _ = strconv.Atoi(r.Form.Get("user_active"))
		err := repo.DB.UpdateUser(user)
		if err != nil {
			log.Println(err)
			ClientError(w, r, http.StatusBadRequest)
			return
		}

		if len(r.Form.Get("password")) > 0 {
			err := repo.DB.UpdatePassword(id, r.Form.Get("password"))
			if err != nil {
				log.Println(err)
				ClientError(w, r, http.StatusBadRequest)
				return
			}
		}
	} else {
		user.Name = r.Form.Get("name")
		user.Email = r.Form.Get("email")
		user.UserActive, _ = strconv.Atoi(r.Form.Get("user_active"))
		user.Password = []byte(r.Form.Get("password"))
		user.AccessLevel = 3

		_, err := repo.DB.InsertUser(user)
		if err != nil {
			log.Println(err)
			ClientError(w, r, http.StatusBadRequest)
			return
		}
	}

}

// DeleteUser adds/edits a user
func (repo *DBRepo) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	_ = repo.DB.DeleteUser(id)
	repo.App.Session.Put(r.Context(), "flash", "User deleted")
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// getPreferenceMapData returns initialized preference data from http request
func getPreferenceMapData(r *http.Request) map[string]string {
	prefMap := make(map[string]string)

	setAData(prefMap, r, "site_url")
	setAData(prefMap, r, "notify_name")
	setAData(prefMap, r, "notify_email")
	setAData(prefMap, r, "smtp_server")
	setAData(prefMap, r, "smtp_port")
	setAData(prefMap, r, "smtp_user")
	setAData(prefMap, r, "smtp_password")
	setAData(prefMap, r, "sms_enabled")
	setAData(prefMap, r, "sms_provider")
	setAData(prefMap, r, "twilio_phone_number")
	setAData(prefMap, r, "twilio_sid")
	setAData(prefMap, r, "twilio_auth_token")
	setAData(prefMap, r, "smtp_from_email")
	setAData(prefMap, r, "smtp_from_name")
	setAData(prefMap, r, "notify_via_sms")
	setAData(prefMap, r, "notify_via_email")
	setAData(prefMap, r, "sms_notify_number")

	if r.Form.Get("sms_enabled") == "0" {
		prefMap["notify_via_sms"] = "0"
	}
	return prefMap
}

// setAData sets a value from request to preference map with key
func setAData(prefMap map[string]string, r *http.Request, key string) {
	prefMap[key] = r.Form.Get(key)
}

// ClientError will display error page for client error i.e. bad request
func ClientError(w http.ResponseWriter, r *http.Request, status int) {
	switch status {
	case http.StatusNotFound:
		show404(w, r)
	case http.StatusInternalServerError:
		show500(w, r)
	default:
		http.Error(w, http.StatusText(status), status)
	}
}

// ServerError will display error page for internal server error
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	_ = log.Output(2, trace)
	show500(w, r)
}

func show404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	http.ServeFile(w, r, "./ui/static/404.html")
}

func show500(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	http.ServeFile(w, r, "./ui/static/500.html")
}

func printTemplateError(w http.ResponseWriter, err error) {
	_, _ = fmt.Fprint(w, fmt.Sprintf(`<small><span class='text-danger'>Error executing template: %s</span></small>`, err))
}
