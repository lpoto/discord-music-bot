package datastore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Datastore struct {
	*log.Logger
	*sql.DB
	config *Configuration
	idx    int
}

type PostgresConfig struct {
	Database string `yaml:"Database" validate:"required"`
	User     string `yaml:"User" validate:"required"`
	Password string `yaml:"Password" validate:"required"`
	Host     string `yaml:"Host" validate:"required"`
	Port     int    `yaml:"Port" validate:"required"`
}

type Configuration struct {
	LogLevel        log.Level       `yaml:"LogLevel" validate:"required"`
	InactiveSongTTL time.Duration   `yaml:"InactiveSongTTL" validate:"required"`
	Postgres        *PostgresConfig `yaml:"Postgres" validate:"required"`
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
	datastore.Info("Oppening datastore connection ...")

	datastore.Debug(
		fmt.Sprintf(
			"Connecting to postgres database: postgres://%s:****@%s:%d/%s",
			datastore.config.Postgres.User,
			datastore.config.Postgres.Host,
			datastore.config.Postgres.Port,
			datastore.config.Postgres.Database,
		),
	)
	db, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			datastore.config.Postgres.Host,
			datastore.config.Postgres.Port,
			datastore.config.Postgres.User,
			datastore.config.Postgres.Password,
			datastore.config.Postgres.Database,
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

	datastore.Info("Datastore connection established")
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

	go datastore.runInactiveSongsCleanup(ctx)

	datastore.Info("Datastore initialized")
	return nil
}

func (datastore *Datastore) getIdx() int {
	i := datastore.idx
	datastore.idx = ((datastore.idx + 1) % 100)
	return i
}
