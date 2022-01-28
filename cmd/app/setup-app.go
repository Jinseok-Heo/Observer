package main

import (
	"flag"
	"fmt"
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/pusher/pusher-http-go"
	"log"
	"net/http"
	"os"
	"server_monitor/internal/channeldata"
	"server_monitor/internal/config"
	"server_monitor/internal/driver"
	"server_monitor/internal/handlers"
	"server_monitor/internal/helpers"
	"time"
)

var (
	db_host = os.Getenv("DB_HOST")
	db_port = os.Getenv("DB_PORT")
	db_user = os.Getenv("DB_USER")
	db_pass = os.Getenv("DB_PASSWORD")
	db_name = os.Getenv("DB_NAME")
	db_ssl  = os.Getenv("DB_SSL")
)

func setupDatabase() (*driver.DB, error) {
	dbHost := flag.String("dbhost", db_host, "database host")
	dbPort := flag.String("dbport", db_port, "database port")
	dbUser := flag.String("dbuser", db_user, "database user")
	dbPass := flag.String("dbpass", db_pass, "database password")
	databaseName := flag.String("db", db_name, "database name")
	dbSsl := flag.String("dbssl", db_ssl, "database ssl setting")

	flag.Parse()

	if *dbUser == "" || *dbHost == "" || *dbPort == "" || *databaseName == "" {
		fmt.Println("Missing required flag.")
		os.Exit(1)
	}

	log.Println("Connecting to database...")
	var dsnString string

	if *dbPass == "" {
		dsnString = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s timezone=UTC connect_timeout=5",
			*dbHost, *dbPort, *dbUser, *databaseName, *dbSsl)
	} else {
		dsnString = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=UTC connect_timeout=5",
			*dbHost, *dbPort, *dbUser, *dbPass, *databaseName, *dbSsl)
	}

	return driver.ConnectMysql(dsnString)
}

func setupSessionManger(db *driver.DB, identifier string, inProduction bool) *scs.SessionManager {
	log.Println("Initializing session manager")
	session = scs.New()
	session.Store = mysqlstore.New(db.SQL)
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.Name = fmt.Sprintf("gbsession_id_%s", identifier)
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = inProduction

	return session
}

func setupMail() chan channeldata.MailJob {
	log.Println("Initializing mail channel and worker pool...")
	mailQueue := make(chan channeldata.MailJob, maxWorkerPoolSize)

	// Start the email dispatcher
	log.Println("Starting email dispatcher...")
	dispatcher := NewDispatcher(mailQueue, maxJobMaxWorkers)
	dispatcher.run()

	return mailQueue
}

func setupPreferenceMap(pusherHost, pusherPort, pusherKey, identifier string) map[string]string {
	log.Println("Getting preferences...")
	preferenceMap = make(map[string]string)

	preferences, err := repo.DB.AllPreferences()
	if err != nil {
		log.Fatal("Cannot read preferences:", err)
	}

	for _, pref := range preferences {
		preferenceMap[pref.Name] = string(pref.Preference)
	}

	preferenceMap["pusher-host"] = pusherHost
	preferenceMap["pusher-port"] = pusherPort
	preferenceMap["pusher-key"] = pusherKey
	preferenceMap["identifier"] = identifier
	preferenceMap["version"] = observerVersion

	return preferenceMap
}

func setupApp() (string, error) {
	insecurePort := *flag.String("port", ":4000", "port to listen on")
	identifier := *flag.String("identifier", "observer", "unique identifier")
	domain := *flag.String("domain", "localhost", "domain name (e.g. example.com)")
	inProduction := *flag.Bool("production", false, "application is in production")

	pusherHost := *flag.String("pusherHost", "", "pusher host")
	pusherPort := *flag.String("pusherPort", "443", "pusher port")
	pusherApp := *flag.String("pusherApp", "9", "pusher app id")
	pusherKey := *flag.String("pusherKey", "", "pusher key")
	pusherSecret := *flag.String("pusherSecret", "", "pusher secret")
	pusherSecure := *flag.Bool("pusherSecure", false, "pusher server uses SSL (true or false)")

	flag.Parse()

	if identifier == "" {
		log.Println("Can't configure identifier.")
		os.Exit(1)
	}

	db, err := setupDatabase()
	if err != nil {
		log.Fatal("Cannot connect to database", err)
	}

	session = setupSessionManger(db, identifier, inProduction)

	mailQueue := setupMail()

	app = config.AppConfig{
		DB:           db,
		Session:      session,
		InProduction: inProduction,
		Domain:       domain,
		PusherSecret: pusherSecret,
		MailQueue:    mailQueue,
		Version:      observerVersion,
		Identifier:   identifier,
	}

	repo = handlers.NewMysqlHandlers(db, &app)
	handlers.NewHandlers(repo, &app)

	app.PreferenceMap = setupPreferenceMap(pusherHost, pusherPort, pusherKey, identifier)

	wsClient = pusher.Client{
		AppID:  pusherApp,
		Secret: pusherSecret,
		Key:    pusherKey,
		Secure: pusherSecure,
		Host:   fmt.Sprintf("%s:%s", pusherHost, pusherPort),
	}

	log.Println("Host", fmt.Sprintf("%s:%s", pusherHost, pusherPort))
	log.Println("Secure", pusherSecure)

	app.WsClient = wsClient

	helpers.NewHelpers(&app)
	return insecurePort, err
}

func createDirIfNotExist(path string) error {
	const mode = 0755

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, mode)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
