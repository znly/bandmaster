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

// TODO(cmc)
type Service struct {
	*bandmaster.ServiceBase // inheritance

	addrs   []string
	timeout time.Duration

	c *memcache.Client
}

// TODO(cmc)
func DefaultConfig() (time.Duration, string) {
	return time.Second, "localhost:11211"
}

// TODO(cmc)
func New(timeout time.Duration, addrs ...string) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(),
		addrs:       addrs,
		timeout:     timeout,
	}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (s *Service) Start(ctx context.Context) error {
	c, err := memcache.New(s.addrs...)
	if err != nil {
		return err
	}
	c.SetTimeout(s.timeout)

	errC := make(chan error, 0)
	go func() {
		if _, err := c.Get("random_key"); err != memcache.ErrCacheMiss {
			_ = c.Close()
			errC <- err
		}
		close(errC)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		if err != nil {
			return err
		}
	}

	s.c = c
	return nil
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func Client(s Service) *memcache.Client { return s.c }
