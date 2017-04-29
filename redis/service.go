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

// DefaultConfig returns a `redis.Pool` with sane defaults.
func DefaultConfig(uri string, opts ...redis.DialOption) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     int(32),
		MaxActive:   int(32),
		IdleTimeout: 0,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(uri, opts...)
			return c, errors.WithStack(err)
		},
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
		ServiceBase: bandmaster.NewServiceBase(),
		pool:        p,
	}
}

// -----------------------------------------------------------------------------

// Start opens a connection and PINGs the server: if everything goes smoothly,
// the service is marked as 'started'; otherwise, an error is returned.
//
// The given context defines the deadline for the above-mentionned operations.
//
// NOTE: Start is used by BandMaster's internal machinery, it shouldn't ever
// have to be called by the end-user of the service.
func (s *Service) Start(
	ctx context.Context, _ map[string]bandmaster.Service,
) error {
	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		c, err := s.pool.Dial()
		if err != nil {
			errC <- err
			return
		}
		defer c.Close()
		if _, err = c.Do("PING"); err != nil {
			errC <- err
			return
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop closes the underlying `redis.Pool`: if everything goes smoothly,
// the service is marked as 'stopped'; otherwise, an error is returned.
//
// The given context defines the deadline for the above-mentionned operations.
//
// NOTE: Start is used by BandMaster's internal machinery, it shouldn't ever
// have to be called by the end-user of the service.
func (s *Service) Stop(ctx context.Context) error {
	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		// If the context gets cancelled (unlikely), this routine will leak
		// until the Close() call actually returns.
		// We don't really care.
		if err := s.pool.Close(); err != nil {
			errC <- err
			return
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		if err != nil {
			return err
		}
	}

	return nil
}

// -----------------------------------------------------------------------------

// Client returns the underlying Redis pool of the given service, or nil if
// it is not actually a `redis.Service`.
func Client(s bandmaster.Service) *redis.Pool {
	ss, ok := s.(*Service)
	if !ok {
		return nil
	}
	return ss.pool
}
