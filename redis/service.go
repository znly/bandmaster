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

// TODO(cmc)
type Service struct {
	*bandmaster.ServiceBase // inheritance

	pool *redis.Pool
}

// TODO(cmc)
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

// TODO(cmc)
func New(p *redis.Pool) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(),
		pool:        p,
	}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
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

// TODO(cmc)
func (s *Service) Stop(ctx context.Context) error {
	errC := make(chan error, 1)
	go func() {
		defer close(errC)
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

// TODO(cmc)
func Client(s Service) *redis.Pool { return s.pool }
