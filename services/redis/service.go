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
	"context"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// Service implements a Redis service based on the 'garyburd/redigo' package.
type Service struct {
	*bandmaster.ServiceBase // inheritance

	pool *redis.Pool
}

// DefaultConfig returns a `redis.Pool` with the following defaults:
//
//   &redis.Pool{
//   	// Maximum number of idle connections in the pool.
//   	MaxIdle: 32,
//   	// Maximum number of connections allocated by the pool at a given time.
//   	// When zero, there is no limit on the number of connections in the pool.
//   	MaxActive: 32,
//   	// Close connections after remaining idle for this duration. If the value
//   	// is zero, then idle connections are not closed. Applications should set
//   	// the timeout to a value less than the server's timeout.
//   	IdleTimeout: 0,
//   	// If Wait is true and the pool is at the MaxActive limit, then Get() waits
//   	// for a connection to be returned to the pool before returning.
//   	Wait: true,
//   	// Dial is an application supplied function for creating and configuring a
//   	// connection.
//   	//
//   	// The connection returned from Dial must not be in a special state
//   	// (subscribed to pubsub channel, transaction started, ...).
//   	Dial: func() (redis.Conn, error) {
//   		c, err := redis.DialURL(uri, opts...)
//   		return c, errors.WithStack(err)
//   	},
//   	// TestOnBorrow is an optional application supplied function for checking
//   	// the health of an idle connection before the connection is used again by
//   	// the application. Argument t is the time that the connection was returned
//   	// to the pool. If the function returns an error, then the connection is
//   	// closed.
//   	TestOnBorrow: func(c redis.Conn, t time.Time) error { return c.Err() },
//   }
//
func DefaultConfig(uri string, opts ...redis.DialOption) *redis.Pool {
	return &redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: 32,
		// Maximum number of connections allocated by the pool at a given time.
		// When zero, there is no limit on the number of connections in the pool.
		MaxActive: 32,
		// Close connections after remaining idle for this duration. If the value
		// is zero, then idle connections are not closed. Applications should set
		// the timeout to a value less than the server's timeout.
		IdleTimeout: 0,
		// If Wait is true and the pool is at the MaxActive limit, then Get() waits
		// for a connection to be returned to the pool before returning.
		Wait: true,
		// Dial is an application supplied function for creating and configuring a
		// connection.
		//
		// The connection returned from Dial must not be in a special state
		// (subscribed to pubsub channel, transaction started, ...).
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(uri, opts...)
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

// New creates a new service using the provided `redis.Pool`.
// Use `DefaultConfig` to get a pre-configured `redis.Pool`.
//
// It doesn't open any connection nor does it do any kind of I/O; i.e. it
// cannot fail.
func New(p *redis.Pool) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // inheritance
		pool:        p,
	}
}

// -----------------------------------------------------------------------------

// Start opens a connection and PINGs the server: if everything goes smoothly,
// the service is marked as 'started'; otherwise, an error is returned.
//
//
// Start is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Start(context.Context, map[string]bandmaster.Service) error {
	c, err := s.pool.Dial()
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err = c.Do("PING"); err != nil {
		return err
	}
	return nil
}

// Stop does nothing.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(context.Context) error {
	return nil
	//return s.pool.Close()
}

// -----------------------------------------------------------------------------

// Client returns the underlying `redis.Pool` of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `redis.Service`.
func Client(s bandmaster.Service) *redis.Pool {
	return s.(*Service).pool // allowed to panic
}
