package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	_ "github.com/joho/godotenv/autoload"

	"github.com/wtran29/go-ecommerce/internal/driver"
	"github.com/wtran29/go-ecommerce/internal/models"
)

const version = "1.0.0"
const cssVersion = "1"

var session *scs.SessionManager

type config struct {
	port int
	env  string
	api  string
	db   struct {
		dsn string
	}
	stripe struct {
		secret string
		key    string
	}
	secretkey string
	frontend  string
}

type application struct {
	config        config
	logger        *slog.Logger
	templateCache map[string]*template.Template
	version       string
	DB            models.DBModel
	Session       *scs.SessionManager
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	app.logger.Info(fmt.Sprintf("Starting HTTP server in %s mode on port %d", app.config.env, app.config.port))
	return srv.ListenAndServe()
}
func main() {
	gob.Register(TransactionData{})
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "Server port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application environment {development|production}")
	flag.StringVar(&cfg.api, "api", "http://localhost:4001", "URL to API")
	flag.StringVar(&cfg.db.dsn, "dsn", fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=disable timezone=UTC connect_timeout=5",
		os.Getenv("ECOMM_HOST"), os.Getenv("ECOMM_PORT"), os.Getenv("ECOMM_USER"), os.Getenv("ECOMM_PW"), os.Getenv("ECOMM_DBNAME")), "DSN")
	flag.StringVar(&cfg.secretkey, "secret", fmt.Sprintf("%v", os.Getenv("SKEY")), "secret key")
	flag.StringVar(&cfg.frontend, "frontend", "http://localhost:4000", "url for frontend")

	flag.Parse()

	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")

	jsonLogger := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonLogger)

	conn, err := driver.OpenDB(cfg.db.dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// set up session
	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Store = postgresstore.New(conn)

	tc := make(map[string]*template.Template)

	app := &application{
		config:        cfg,
		logger:        logger,
		templateCache: tc,
		version:       version,
		DB:            models.DBModel{DB: conn},
		Session:       session,
	}

	go app.ListenToWsChannel()

	// app.DB.CreateTables()

	err = app.serve()
	if err != nil {
		app.logger.Error("Error starting http server", err)
		log.Fatal(err)
	}

}
