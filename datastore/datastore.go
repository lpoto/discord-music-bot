package datastore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Datastore struct {
	*log.Logger
	*sql.DB
}

type Configuration struct {
	Database string `yaml:"Database" validate:"required"`
	Host     string `yaml:"Host" validate:"required"`
	Port     int    `yaml:"Port" validate:"required"`
	User     string `yaml:"User" validate:"required"`
	Password string `yaml:"Password" validate:"required"`
}

// NewDatastore constructs an object that handles persisting
// the models to the postgres database and recieving them from it.
// It does not implement any of the bot's logics.
func NewDatastore(logLevel log.Level) *Datastore {
	l := log.New()
	l.SetLevel(logLevel)
	l.Debug("Datastore created")
	return &Datastore{Logger: l}
}

// Connect opens a new postges connection based on the
// provided database configuration
func (datastore *Datastore) Connect(configuration *Configuration) error {
	datastore.Info("Oppening postgres connection ...")

	db, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			configuration.Host,
			configuration.Port,
			configuration.User,
			configuration.Password,
			configuration.Database,
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
	if err := datastore.createSongTable(); err != nil {
		return err
	}

	datastore.Info("Datastore initialized")
	return nil
}

func (datastore *Datastore) escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", `"`)
}
