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

package memcached

import (
	"context"
	"time"

	"github.com/rainycape/memcache"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// Service implements a Memcached service based on the 'rainycape/memcached'
// package.
type Service struct {
	*bandmaster.ServiceBase // inheritance

	addrs   []string
	timeout time.Duration

	c *memcache.Client
}

type Config struct {
	Addrs     []string
	RWTimeout time.Duration
}

// DefaultConfig returns a `Config` with timeout of 1 second.
func DefaultConfig(addrs ...string) Config {
	return Config{Addrs: addrs, RWTimeout: time.Second}
}

// New creates a new service using the provided parameters.
// Use `DefaultConfig()` or the helpers for environment-based configuration to
// get a pre-configured `Config`.
//
// It doesn't open any connection nor does it do any kind of I/O; i.e. it
// cannot fail.
func New(c Config) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // inheritance
		addrs:       c.Addrs,
		timeout:     c.RWTimeout,
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

	var err error
	if s.c == nil { // idempotency
		s.c, err = memcache.New(s.addrs...)
		if err != nil {
			return err
		}
		s.c.SetTimeout(s.timeout)
		if _, err := s.c.Get("random_key"); err != memcache.ErrCacheMiss {
			_ = s.Stop(context.Background())
			return err
		}
	}
	return nil
}

// Stop closes the underlying `memcache.Client`: if everything goes smoothly,
// the service is marked as 'stopped'; otherwise, an error is returned.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(ctx context.Context) error {
	if s.c != nil {
		if err := s.c.Close(); err != nil {
			return err
		}
		s.c = nil // idempotency & restart support
	}
	return nil
}

// -----------------------------------------------------------------------------

// Client returns the underlying `memcache.Client` of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `nats.Service`.
func Client(s bandmaster.Service) *memcache.Client {
	return s.(*Service).c // allowed to panic
}
