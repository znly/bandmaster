// Copyright © 2017 Zenly <hello@zen.ly>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sql

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// Env can be used to configure a SQL session via the environment.
//
// It comes with sane defaults for a local development set-up.
// The default driver is setup to Postgres with a working local Addr
type Env struct {
	DriverName      string        `envconfig:"DRIVER_NAME" default:"postgres"`
	Addr            string        `envconfig:"ADDR" default:"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"`
	ConnMaxLifetime time.Duration `envconfig:"CONN_MAX_LIFETIME" default:"10m"`
	MaxIdleConns    int           `envconfig:"MAX_IDLE_CONNS" default:"1"`
	MaxOpenConns    int           `envconfig:"MAX_OPEN_CONNS" default:"2"`
}

// NewEnv parses the environment and returns a new `Env` structure.
//
// `prefix` defines the prefix for the environment keys, e.g. with a 'XX' prefix,
// 'REPLICAS' would become 'XX_REPLICAS'.
func NewEnv(prefix string) (*Env, error) {
	e := &Env{}
	if err := envconfig.Process(prefix, e); err != nil {
		return nil, errors.WithStack(err)
	}
	return e, nil
}

// Config returns a `Config` using the values from the environment.
func (e *Env) Config() *Config {
	var config Config
	config.DriverName = e.DriverName
	config.Addr = e.Addr
	config.ConnMaxLifetime = e.ConnMaxLifetime
	config.MaxIdleConns = e.MaxIdleConns
	config.MaxOpenConns = e.MaxOpenConns
	return &config
}
