package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/wtran29/go-ecommerce/internal/driver"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
	stripe struct {
		secret string
		key    string
	}
}

type application struct {
	config  config
	logger  *slog.Logger
	version string
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

	app.logger.Info(fmt.Sprintf("Starting backend server in %s mode on port %d", app.config.env, app.config.port))
	return srv.ListenAndServe()
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4001, "Server port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application environment {development|production|maintenance}")
	flag.StringVar(&cfg.db.dsn, "dsn", "host=localhost port=5432 user=postgres password=postgres dbname=ecomm_db sslmode=disable timezone=UTC connect_timeout=5", "DSN")

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

	app := &application{
		config:  cfg,
		logger:  logger,
		version: version,
	}

	err = app.serve()
	if err != nil {
		app.logger.Error("Error starting backend server", err)
		log.Fatal(err)
	}
}
