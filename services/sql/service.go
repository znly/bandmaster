package sql

import (
	"context"
	"database/sql"
	"time"

	"github.com/znly/bandmaster"
)

type Service struct {
	*bandmaster.ServiceBase // "inheritance"

	config *Config
	db     *sql.DB
}

type Config struct {
	DriverName      string
	Addr            string
	ConnMaxLifetime time.Duration
	MaxIdleConns    int
	MaxOpenConns    int
}

// New creates a new SQL service using the provided `Config`.
// You may use the helpers for environment-based configuration to get a
// pre-configured `Config` with sane defaults.
//
// It doesn't open any connection nor does it do any kind of I/O; i.e. it
// cannot fail.
func New(conf *Config) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // "inheritance"
		config:      conf,
		db:          nil,
	}
}

// -----------------------------------------------------------------------------

// Start opens PING the server and establish a connection if necessary:
// the service is marked as 'started' in case of success;
// otherwise, an error is returned.
//
//
// Start is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Start(context.Context, map[string]bandmaster.Service) error {
	var err error
	s.db, err = sql.Open(s.config.DriverName, s.config.Addr)
	if err != nil {
		return err
	}
	s.db.SetConnMaxLifetime(s.config.ConnMaxLifetime)
	s.db.SetMaxIdleConns(s.config.MaxIdleConns)
	s.db.SetMaxOpenConns(s.config.MaxOpenConns)

	if err := s.db.Ping(); err != nil {
		return err
	}
	return nil
}

// Stop closes the database, releasing any open resources.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(context.Context) error {
	return s.db.Close()
}

// -----------------------------------------------------------------------------

// Client returns the underlying `sql.DB` of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `sql.DB`.
func Client(s bandmaster.Service) *sql.DB {
	return s.(*Service).db
}
