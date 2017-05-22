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

package cql

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// Service implements a CQL service based on the 'gocql/gocql' package.
type Service struct {
	*bandmaster.ServiceBase // inheritance

	cc *gocql.ClusterConfig
	s  *gocql.Session
}

// DefaultConfig returns a `gocql.ClusterConfig` with the following defaults:
//
//   &gocql.ClusterConfig{
//   	// connection timeout
//   	Timeout: time.Second * 30,
//   	// initial connection timeout, used during initial dial to server
//   	ConnectTimeout: time.Second * 30,
//   	// number of connections per host
//   	NumConns: 8,
//   	// default consistency level
//   	Consistency: gocql.LocalQuorum,
//   }
//
func DefaultConfig(addrs ...string) *gocql.ClusterConfig {
	cluster := gocql.NewCluster(addrs...)
	// default consistency level
	cluster.Consistency = gocql.LocalQuorum
	// number of connections per host
	cluster.NumConns = 8
	// initial connection timeout, used during initial dial to server
	cluster.ConnectTimeout = time.Second * 30
	// connection timeout
	cluster.Timeout = time.Second * 30

	return cluster
}

// New creates a new service using the provided `gocql.ClusterConfig`.
// Use `DefaultConfig()` or the helpers for environment-based configuration to
// get a pre-configured `gocql.ClusterConfig`.
//
// It doesn't open any connection nor does it do any kind of I/O; i.e. it
// cannot fail.
func New(cc *gocql.ClusterConfig) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // inheritance
		cc:          cc,
	}
}

// -----------------------------------------------------------------------------

// Start opens a connection and requests the version of the server: if
// everything goes smoothly, the service is marked as 'started'; otherwise, an
// error is returned.
//
// The given context defines the deadline for the above-mentionned operations.
//
// Start is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Start(
	ctx context.Context, _ map[string]bandmaster.Service,
) error {
	var err error
	if s.s == nil {
		s.s, err = s.cc.CreateSession()
		if err != nil {
			return err
		}
		if err = s.s.Query(
			"SELECT cql_version FROM system.local",
		).WithContext(ctx).Exec(); err != nil {
			_ = s.Stop(context.Background())
			return err
		}
	}
	return nil
}

// Stop closes the underlying `gocql.Session`: if everything goes smoothly,
// the service is marked as 'stopped'; otherwise, an error is returned.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(ctx context.Context) error {
	if s.s != nil {
		s.s.Close()
		s.s = nil // idempotency & restart support
	}
	return nil
}

// -----------------------------------------------------------------------------

// Client returns the underlying `gocql.Session` of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `cql.Service`.
func Client(s bandmaster.Service) *gocql.Session {
	return s.(*Service).s // allowed to panic
}

// Config returns the underlying `gocql.ClusterConfig` of the given service.
//
// NOTE: This will panic if `s` is not a `cql.Service`.
func Config(s bandmaster.Service) *gocql.ClusterConfig {
	return s.(*Service).cc // allowed to panic
}
