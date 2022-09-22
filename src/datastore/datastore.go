package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Datastore struct {
	*log.Logger
	*sql.DB
	config *Configuration
	idx    int
}

type Configuration struct {
	LogLevel log.Level `yaml:"LogLevel" validate:"required"`
}

// NewDatastore constructs an object that handles persisting
// the models to the postgres database and recieving them from it.
// It does not implement any of the bot's logics.
func NewDatastore(config *Configuration) *Datastore {
	l := log.New()
	l.SetLevel(config.LogLevel)
	l.Debug("Datastore created")
	return &Datastore{Logger: l, config: config, idx: 0}
}

// Connect opens a new postges connection based on the
// provided database configuration
func (datastore *Datastore) Connect() error {
	datastore.Info("Oppening postgres connection ...")

	host := os.Getenv("POSTGRES_HOST")
	if len(host) == 0 {
		return errors.New("Missing environment variable 'POSTGRES_HOST'")
	}
	portString := os.Getenv("POSTGRES_PORT")
	if len(portString) == 0 {
		return errors.New("Missing environment variable 'POSTGRES_PORT'")
	}
	user := os.Getenv("POSTGRES_USER")
	if len(user) == 0 {
		return errors.New("Missing environment variable 'POSTGRES_USER'")
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if len(password) == 0 {
		return errors.New("Missing environment variable 'POSTGRES_PASSWORD'")
	}
	database := os.Getenv("POSTGRES_DB")
	if len(database) == 0 {
		return errors.New("Missing environment variable 'POSTGRES_DB'")
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		return errors.New("'POSTGRES_PORT' is not a valid port number")
	}

	db, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host,
			port,
			user,
			password,
			database,
		),
	)
	if err != nil {
		return err
	}
	// NOTE: ping the databse so we make sure there is a valid connection
	if err := db.Ping(); err != nil {
		return err
	}
	datastore.DB = db

	datastore.Info("Postgres connection established")
	return nil
}

// Init creates all the tables required by the datastore
// and runs the goroutine required for deleting the
// outdated inactive songs.
func (datastore *Datastore) Init(ctx context.Context) error {
	datastore.Debug("Initializing datastore ...")

	if err := datastore.createQueueTable(); err != nil {
		return err
	}
	if err := datastore.createQueueOptionTable(); err != nil {
		return err
	}
	if err := datastore.createSongTable(); err != nil {
		return err
	}
	if err := datastore.createInactiveSongTable(); err != nil {
		return err
	}

	datastore.Info("Datastore initialized")
	return nil
}

func (datastore *Datastore) getIdx() int {
	i := datastore.idx
	datastore.idx = ((datastore.idx + 1) % 100)
	return i
}

func (datastore *Datastore) escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "`")
}

func (datastore *Datastore) unescapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "`", "'")
}
