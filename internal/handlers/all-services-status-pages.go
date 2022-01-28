package handlers

import (
	"net/http"
	"server_monitor/internal/helpers"
)

// AllHealthyServices lists all healthy services
func (repo *DBRepo) AllHealthyServices(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "healthy", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

func (repo *DBRepo) AllWarningsServices(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "warning", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

func (repo *DBRepo) AllProblemServices(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "problems", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}

func (repo *DBRepo) AllPendingServices(w http.ResponseWriter, r *http.Request) {
	err := helpers.RenderPage(w, r, "pending", nil, nil)
	if err != nil {
		printTemplateError(w, err)
	}
}
