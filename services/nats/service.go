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

	"github.com/nats-io/nats"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// Service implements a NATS service based on the 'nats-io/nats' package.
type Service struct {
	*bandmaster.ServiceBase // inheritance

	opts *nats.Options
	c    *nats.Conn
}

// DefaultConfig returns a `nats.Options` with the following defaults:
//
//   &nats.Options {
//   	Servers:          addrs,
//   	AllowReconnect:   true,
//   	MaxReconnect:     DefaultMaxReconnect,
//   	ReconnectWait:    DefaultReconnectWait,
//   	Timeout:          DefaultTimeout,
//   	PingInterval:     DefaultPingInterval,
//   	MaxPingsOut:      DefaultMaxPingOut,
//   	SubChanLen:       DefaultMaxChanLen,
//   	ReconnectBufSize: DefaultReconnectBufSize,
//   	Dialer: &net.Dialer{
//   		Timeout: DefaultTimeout,
//   	},
//   }
//
// With:
//
//   DefaultPort             = 4222
//   DefaultMaxReconnect     = 60
//   DefaultReconnectWait    = 2 * time.Second
//   DefaultTimeout          = 2 * time.Second
//   DefaultPingInterval     = 2 * time.Minute
//   DefaultMaxPingOut       = 2
//   DefaultMaxChanLen       = 8192            // 8k
//   DefaultReconnectBufSize = 8 * 1024 * 1024 // 8MB
//
func DefaultConfig(addrs ...string) *nats.Options {
	natsOpts := nats.DefaultOptions
	natsOpts.Servers = addrs
	return &natsOpts
}

// New creates a new service using the provided `nats.Options`.
// Use `DefaultConfig` to get a pre-configured `nats.Options`.
//
// It doesn't open any connection nor does it do any kind of I/O; i.e. it
// cannot fail.
func New(opts *nats.Options) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // inheritance
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
	}
	if s.c.Status() != nats.CONNECTED {
		return errors.New("nats: connection failure") // TODO(cmc): typed error
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
// NOTE: This will panic if `s` is not a `kafka.Service`.
func Client(s bandmaster.Service) *nats.Conn {
	return s.(*Service).c // allowed to panic
}
