package db

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/iancoleman/strcase"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Config has all the values we need to connect to a DB
type Config struct {
	// The user to use when connecting
	User     string
	Password string
	Host     string
	Port     int
	// The name of the DB to connect to
	Name string
}

// DB is our local DB struct
type DB struct {
	*sqlx.DB
}

func open(c *cli.Context, name string) (*DB, error) {

	var port int
	var host string
	if c == nil {
		port = 5433
		host = "localhost"
	} else {
		port = c.Int("db.port")
		host = c.String("db.host")
	}

	conf := Config{
		User:     "txnotify",
		Password: "password",
		Port:     port,
		Host:     host,
		Name:     "txnotify",
	}

	// Define SSL mode.
	sslMode := "disable" // require

	// Query parameters.
	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")
	q.Set("fallback_application_name", "txnotify")

	databaseHostWithPort := conf.Host + ":" + strconv.Itoa(conf.Port)
	databaseURL := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			conf.User,
			conf.Password,
		),
		Host:     databaseHostWithPort,
		Path:     conf.Name,
		RawQuery: q.Encode(),
	}

	database, err := sqlx.Open("postgres", databaseURL.String())
	if err != nil {
		return nil, fmt.Errorf("cannot connect to database %s with user %s at %s: %w",
			conf.Name,
			conf.User,
			databaseHostWithPort,
			err,
		)
	}

	// just in case it isnt created yet
	_, _ = database.Exec("CREATE DATABASE " + name)

	conf = Config{
		User:     "txnotify",
		Password: "password",
		Port:     port,
		Host:     host,
		Name:     name,
	}

	// Define SSL mode.
	sslMode = "disable" // require

	// Query parameters.
	q = make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")
	q.Set("fallback_application_name", "txnotify")

	databaseHostWithPort = conf.Host + ":" + strconv.Itoa(conf.Port)
	databaseURL = url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			conf.User,
			conf.Password,
		),
		Host:     databaseHostWithPort,
		Path:     conf.Name,
		RawQuery: q.Encode(),
	}

	database, err = sqlx.Open("postgres", databaseURL.String())
	if err != nil {
		return nil, fmt.Errorf("cannot connect to database %s with user %s at %s: %w",
			conf.Name,
			conf.User,
			databaseHostWithPort,
			err,
		)
	}

	log := log.WithFields(logrus.Fields{
		"user":     conf.User,
		"host":     databaseHostWithPort,
		"database": conf.Name,
	})
	log.Debug("Opening connection to DB")

	err = database.QueryRow("SELECT FROM pg_database WHERE datname=$1", conf.Name).Scan()
	switch {
	case err == nil: // database exists
	case errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "does not exist"): // database does not exist
		if _, err := database.Exec("CREATE DATABASE " + conf.Name); err != nil {
			return nil, fmt.Errorf("cannot create database %s: %v", conf.Name, err)
		}

		if _, err := database.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s",
			conf.Name, conf.User)); err != nil {
			return nil, fmt.Errorf("cannot grant privileges to test user (%s): %v", conf.User, err)
		}
	default:
		return nil, err
	}

	return &DB{
		DB: database,
	}, nil
}

// GetDatabaseConfig returns a DB config suitable for testing purposes. The
// given argument is added to the name of the database
func New(c *cli.Context, name ...string) (*DB, error) {
	if len(name) == 0 {
		name = []string{""}
	} else {
		name[0] = "_" + name[0]
	}

	database, err := open(c, "txnotify"+strings.ToLower(name[0]))
	if err != nil {
		return nil, err
	}

	// the query beneath is the first thing that happens when we connect. we might then run into an
	// issue where Postgres is still starting up. we therefore retry this multiple times
	var pgVersion string
	const max = 5
	for i := 0; i < max; i++ {
		const sleep = time.Second * 2
		// language=pgsql
		const query = `SHOW SERVER_VERSION`
		err = database.QueryRowx(query).Scan(&pgVersion)
		pqErr := new(pq.Error)
		switch {
		case err == nil:
			break
		case errors.As(err, &pqErr) &&
			pqErr.Severity == pq.Efatal && pqErr.Message != "the database system is starting up":
			return nil, err
		default:
			if i != max-1 {
				log.WithError(err).WithField("sleep", sleep).Debug("Could not query Postgres, trying again")
			}
			time.Sleep(sleep)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("could not get Postgres version: %w", err)
	}

	err = database.MigrateUp()
	if err != nil {
		return nil, err
	}

	log.Info("Opened connection to DB")

	return database, err
}

// MigrateUp migrates everything up
func (d *DB) MigrateUp() error {
	start := time.Now()
	log.Debug("Starting DB migration")

	driver, err := postgres.WithInstance(d.DB.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "postgres", driver)
	if err != nil {
		m, err = migrate.NewWithDatabaseInstance("file://../db/migrations", "postgres", driver)
		if err != nil {
			return fmt.Errorf("could not create driver")
		}
	}

	// Migrate all the way up ...
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Info("No migrations applied")
		} else {
			log.WithError(err).Error("Could not migrate up")
			return fmt.Errorf("could not migrate up: %w", err)
		}
	}

	log.WithField("duration", time.Since(start)).Info("Successfully migrated up")
	return nil
}

// CreateMigration creates a new empty migration file with correct name
func (d *DB) CreateMigration(name string) (string, error) {
	migrationTime := time.Now().UTC().Format("200601021504")
	baseFileName := migrationTime + "_" + strcase.ToSnake(name)

	// TODO: Get current dir. Not of the file, but the base dir of the program
	fileNameUp := path.Join(
		"migrations",
		baseFileName+".up.pgsql")
	if _, err := os.Create(fileNameUp); err != nil {
		return "", err
	}

	fileNameDown := path.Join(
		"migrations",
		baseFileName+".down.pgsql")
	fmt.Println(fileNameDown)
	if _, err := os.Create(fileNameDown); err != nil {
		return "", err
	}
	return baseFileName, nil
}

func NewTest(name string) *DB {
	database, err := New(nil, name)
	if err != nil {
		panic(fmt.Errorf("could not create new: %w", err))
	}

	defer func() {
		// just in case it isnt created yet
		_, _ = database.Exec("DROP DATABASE " + name)
		// just in case it isnt created yet
		_, _ = database.Exec("UPDATE schema_migrations SET version = 0, dirty = f")
	}()

	return database
}
