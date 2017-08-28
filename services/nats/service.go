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

package nats

import (
	"context"
	"errors"

	"github.com/nats-io/go-nats"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// Service implements a NATS service based on the 'nats-io/nats' package.
type Service struct {
	*bandmaster.ServiceBase // "inheritance"

	opts *nats.Options
	c    *nats.Conn
}

// New creates a new NATS service using the provided `nats.Options`.
// You may use the helpers for environment-based configuration to get a
// pre-configured `nats.Options` with sane defaults.
//
// It doesn't open any connection nor does it do any kind of I/O; i.e. it
// cannot fail.
func New(opts *nats.Options) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // "inheritance"
		opts:        opts,
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
		s.c, err = s.opts.Connect()
		if err != nil {
			return err
		}
		if s.c.Status() != nats.CONNECTED {
			_ = s.Stop(context.Background())
			return errors.New("nats: connection failure") // TODO(cmc): typed error
		}
	}
	return nil
}

// Stop closes the underlying `nats.Conn`: if everything goes smoothly,
// the service is marked as 'stopped'; otherwise, an error is returned.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(ctx context.Context) error {
	if s.c != nil {
		s.c.Close()
		s.c = nil // idempotency & restart support
	}
	return nil
}

// -----------------------------------------------------------------------------

// Client returns the underlying `nats.Conn` of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `nats.Service`.
func Client(s bandmaster.Service) *nats.Conn {
	return s.(*Service).c // allowed to panic
}

// Config returns the underlying `nats.Options` of the given service.
//
// NOTE: This will panic if `s` is not a `nats.Service`.
func Config(s bandmaster.Service) *nats.Options {
	return s.(*Service).opts // allowed to panic
}
