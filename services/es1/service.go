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

package es

import (
	"context"

	elastic "gopkg.in/olivere/elastic.v2"

	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// Service implements an ElasticSearch (v1) service based on the
// 'olivere/elastic.v1' package.
type Service struct {
	*bandmaster.ServiceBase // "inheritance"

	conf *Config

	c *elastic.Client
}

// Config contains the necessary configuration for an ElasticSearch (v1) service.
type Config struct {
	Addr string
	Opts []elastic.ClientOptionFunc
}

// New creates a new ElasticSearch (v1) service using the provided configuration.
// You may use the helpers for environment-based configuration to get a
// pre-configured `Config` with sane defaults.
//
// It doesn't open any connection nor does it do any kind of I/O; i.e. it
// cannot fail.
func New(conf *Config) bandmaster.Service {
	conf.Opts = append(conf.Opts, elastic.SetURL(conf.Addr))
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // "inheritance"
		conf:        conf,
	}
}

// -----------------------------------------------------------------------------

// Start opens a connection and pings the server: if everything goes smoothly,
// the service is marked as 'started'; otherwise, an error is returned.
//
//
// Start is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Start(
	_ context.Context, _ map[string]bandmaster.Service,
) error {
	var err error
	if s.c == nil {
		s.c, err = elastic.NewClient(s.conf.Opts...)
		if err != nil {
			return err
		}
		_, _, err := s.c.Ping().URL(s.conf.Addr).Timeout("1s").Do()
		if err != nil {
			_ = s.Stop(context.Background())
			return err
		}
	}
	return nil
}

// Stop closes the underlying `elastic.Client`: if everything goes smoothly,
// the service is marked as 'stopped'; otherwise, an error is returned.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(ctx context.Context) error {
	if s.c != nil {
		s.c = nil // idempotency & restart support
	}
	return nil
}

// -----------------------------------------------------------------------------

// Client returns the underlying `elastic.Client` of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `es.Service`.
func Client(s bandmaster.Service) *elastic.Client {
	return s.(*Service).c // allowed to panic
}
