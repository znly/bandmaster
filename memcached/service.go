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

	"github.com/pkg/errors"
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
func New(timeout time.Duration, addrs ...string) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(),
		addrs:       addrs,
		timeout:     timeout,
	}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (s *Service) Run(
	_, lifeCtx context.Context,
) (<-chan error, <-chan error) {
	bootErrC := make(chan error, 1)
	lifeErrC := make(chan error, 1)

	go func() {
		defer close(lifeErrC) // stopped

		c, err := memcache.New(s.addrs...)
		if err != nil {
			bootErrC <- errors.Wrap(err, "couldn't start memcached client")
			return
		}
		c.SetTimeout(s.timeout)
		s.c = c

		if _, err := c.Get("random_key"); err != memcache.ErrCacheMiss {
			bootErrC <- errors.Wrap(err, "couldn't start memcached client")
			return
		}
		close(bootErrC) // started

		select {
		case <-lifeCtx.Done():
			lifeErrC <- errors.WithStack(lifeCtx.Err())
			return
		}
	}()

	return bootErrC, lifeErrC
}
