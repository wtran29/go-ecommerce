package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

const version = "1.0.0"

type config struct {
	port int
	smtp struct {
		host     string
		port     int
		username string
		password string
	}

	frontend string
}

type application struct {
	config  config
	logger  *slog.Logger
	version string
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 5000, "Server port to listen on")
	flag.StringVar(&cfg.smtp.host, "smtphost", os.Getenv("SMTPHOST"), "smtp host")
	flag.StringVar(&cfg.smtp.username, "smtpuser", os.Getenv("SMTPUSER"), "smtp user")
	flag.StringVar(&cfg.smtp.password, "smtppass", os.Getenv("SMTPPW"), "smtp password")
	smptport, _ := strconv.Atoi(os.Getenv("SMTPPORT"))
	flag.IntVar(&cfg.smtp.port, "smtpport", smptport, "smtp port")
	flag.StringVar(&cfg.frontend, "frontend", "http://localhost:4000", "url for frontend")

	flag.Parse()

	jsonLogger := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonLogger)

	app := &application{
		config:  cfg,
		logger:  logger,
		version: version,
	}

	app.CreateDirIfNotExist("./invoices")

	err := app.serve()
	if err != nil {
		app.logger.Error("Error starting backend server", err)
		log.Fatal(err)
	}
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

	app.logger.Info(fmt.Sprintf("Starting invoice microservice on port %d", app.config.port))
	return srv.ListenAndServe()
}
