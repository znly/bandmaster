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

package redis

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/fatih/camelcase"
	"github.com/garyburd/redigo/redis"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// Env can be used to configure a Redis session via the environment.
//
// It comes with sane default for a local development set-up.
type Env struct {
	URI            string        `envconfig:"URI" default:"redis://localhost:6379"`
	MaxConnsIdle   int           `envconfig:"MAX_CONNS_IDLE" default:"32"`
	MaxConnsActive int           `envconfig:"MAX_CONNS_ACTIVE" default:"32"`
	TimeoutIdle    time.Duration `envconfig:"TIMEOUT_IDLE" default:"45s"`
	TimeoutRead    time.Duration `envconfig:"TIMEOUT_READ" default:"2s"`
	TimeoutWrite   time.Duration `envconfig:"TIMEOUT_WRITE" default:"2s"`
	Wait           bool          `envconfig:"WAIT" default:"true"`
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

// Config returns a `redis.Pool` using the values from the environment.
func (e *Env) Config() *redis.Pool {
	return &redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: e.MaxConnsIdle,
		// Maximum number of connections allocated by the pool at a given time.
		// When zero, there is no limit on the number of connections in the pool.
		MaxActive: e.MaxConnsActive,
		// Close connections after remaining idle for this duration. If the value
		// is zero, then idle connections are not closed. Applications should set
		// the timeout to a value less than the server's timeout.
		IdleTimeout: e.TimeoutIdle,
		// If Wait is true and the pool is at the MaxActive limit, then Get() waits
		// for a connection to be returned to the pool before returning.
		Wait: e.Wait,
		// Dial is an application supplied function for creating and configuring a
		// connection.
		//
		// The connection returned from Dial must not be in a special state
		// (subscribed to pubsub channel, transaction started, ...).
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(e.URI,
				redis.DialReadTimeout(e.TimeoutRead),
				redis.DialWriteTimeout(e.TimeoutWrite),
			)
			return c, errors.WithStack(err)
		},
		// TestOnBorrow is an optional application supplied function for checking
		// the health of an idle connection before the connection is used again by
		// the application. Argument t is the time that the connection was returned
		// to the pool. If the function returns an error, then the connection is
		// closed.
		TestOnBorrow: func(c redis.Conn, t time.Time) error { return c.Err() },
	}
}

// -----------------------------------------------------------------------------

func (e *Env) String() string {
	cv := reflect.ValueOf(*e)
	fields := reflect.TypeOf(*e)
	fieldStrs := make([]string, fields.NumField())
	for i := 0; i < fields.NumField(); i++ {
		f := fields.Field(i)
		if len(f.Name) > 0 && strings.ToLower(f.Name[:1]) == f.Name[:1] {
			continue // private field
		}
		fNameParts := camelcase.Split(f.Name)
		for i, fnp := range fNameParts {
			fNameParts[i] = strings.ToUpper(fnp)
		}
		fName := strings.Join(fNameParts, "_")
		itf := cv.Field(i).Interface()
		if _, ok := itf.(fmt.Stringer); ok {
			fieldStrs[i] = fmt.Sprintf("%s = %s", fName, itf)
		} else {
			fieldStrs[i] = fmt.Sprintf("%s = %v", fName, itf)
		}
	}
	return strings.Join(fieldStrs, "\n") + "\n"
}
