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
	"github.com/wtran29/go-ecommerce/internal/driver"
	"github.com/wtran29/go-ecommerce/internal/models"
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
	smtp struct {
		host     string
		port     int
		username string
		password string
	}
	secretkey string
	frontend  string // address for front end
}

type application struct {
	config  config
	logger  *slog.Logger
	version string
	DB      models.DBModel
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
	flag.StringVar(&cfg.db.dsn, "dsn", fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=disable timezone=UTC connect_timeout=5",
		os.Getenv("ECOMM_HOST"), os.Getenv("ECOMM_PORT"), os.Getenv("ECOMM_USER"), os.Getenv("ECOMM_PW"), os.Getenv("ECOMM_DBNAME")), "DSN")
	flag.StringVar(&cfg.smtp.host, "smtphost", os.Getenv("SMTPHOST"), "smtp host")
	flag.StringVar(&cfg.smtp.username, "smtpuser", os.Getenv("SMTPUSER"), "smtp user")
	flag.StringVar(&cfg.smtp.password, "smtppass", os.Getenv("SMTPPW"), "smtp password")
	smptport, _ := strconv.Atoi(os.Getenv("SMTPPORT"))
	flag.IntVar(&cfg.smtp.port, "smtpport", smptport, "smtp port")
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

	app := &application{
		config:  cfg,
		logger:  logger,
		version: version,
		DB:      models.DBModel{DB: conn},
	}

	// encryptKey, _ := app.GenerateEncryptionKey(16)
	// fmt.Println(encryptKey)

	err = app.serve()
	if err != nil {
		app.logger.Error("Error starting backend server", err)
		log.Fatal(err)
	}
}
