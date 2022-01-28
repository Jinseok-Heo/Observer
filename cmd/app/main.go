package main

import (
	"encoding/gob"
	"github.com/alexedwards/scs/v2"
	"github.com/pusher/pusher-http-go"
	"log"
	"net/http"
	"os"
	"runtime"
	"server_monitor/internal/config"
	"server_monitor/internal/handlers"
	"server_monitor/internal/models"
	"time"
)

var preferenceMap map[string]string
var app config.AppConfig
var session *scs.SessionManager
var wsClient pusher.Client
var repo *handlers.DBRepo

const observerVersion = "1.0.0"
const maxWorkerPoolSize = 5
const maxJobMaxWorkers = 5

func init() {
	gob.Register(models.User{})
	_ = os.Setenv("TZ", "Asia/Seoul")
}

func main() {
	insecurePort, err := setupApp()
	if err != nil {
		log.Fatal(err)
	}

	defer close(app.MailQueue)
	defer app.DB.SQL.Close()

	log.Printf("******************************************")
	log.Printf("** %sVigilate%s v%s built in %s", "\033[31m", "\033[0m", observerVersion, runtime.Version())
	log.Printf("**----------------------------------------")
	log.Printf("** Running with %d Processors", runtime.NumCPU())
	log.Printf("** Running on %s", runtime.GOOS)
	log.Printf("******************************************")

	srv := &http.Server{
		Addr:              insecurePort,
		Handler:           routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	log.Printf("Starting HTTP server on port %s....", insecurePort)

	if err = srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
