package dbrepo

import (
	"database/sql"
	"server_monitor/internal/config"
	"server_monitor/internal/repository"
)

var app *config.AppConfig

type mysqlDBRepo struct {
	App *config.AppConfig
	DB  *sql.DB
}

func NewMysqlRepo(Conn *sql.DB, a *config.AppConfig) repository.DatabaseRepo {
	app = a
	return &mysqlDBRepo{
		App: a,
		DB:  Conn,
	}
}
