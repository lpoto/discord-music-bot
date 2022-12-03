package datastore

import (
	"context"
	"database/sql"
	"discord-music-bot/datastore/queue"
	"discord-music-bot/datastore/song"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Datastore struct {
	*log.Logger
	config *Configuration
	queue  *queue.QueueStore
	song   *song.SongStore
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
	return &Datastore{Logger: l, config: config}
}

// Connect opens a new postges connection based on the
// provided database configuration
func (datastore *Datastore) Connect() error {
	datastore.Info("Oppening datastore connection ...")

	datastore.WithField("Url",
		fmt.Sprintf(
			"postgres://%s:****@%s:%d/%s",
			datastore.config.Postgres.User,
			datastore.config.Postgres.Host,
			datastore.config.Postgres.Port,
			datastore.config.Postgres.Database,
		),
	).Debug("Connecting to postgres database")
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
	datastore.queue = queue.NewQueueStore(db, datastore.Logger)
	datastore.song = song.NewSongStore(
		db,
		datastore.Logger,
		datastore.config.InactiveSongTTL,
	)

	datastore.Info("Datastore connection established")
	return nil
}

// Init creates all the tables required by the datastore
// and runs the goroutine required for deleting the
// outdated inactive songs.
func (datastore *Datastore) Init(ctx context.Context, runInactiveSongsCleanup bool) error {
	datastore.Debug("Initializing datastore ...")

	if err := datastore.queue.Init(); err != nil {
		return err
	}
	if err := datastore.song.Init(); err != nil {
		return err
	}

	if runInactiveSongsCleanup {
		go datastore.song.RunInactiveSongsCleanup(ctx)
	}

	datastore.Info("Datastore initialized")
	return nil
}

// Queue returns the object that handles persisting and
// removing Queues in the datastore.
func (datastore *Datastore) Queue() *queue.QueueStore {
	return datastore.queue
}

// Song returns the object that handles persisting and
// removing Songs in the datastore.
func (datastore *Datastore) Song() *song.SongStore {
	return datastore.song
}
