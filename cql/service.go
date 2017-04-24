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
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type Service struct {
	*bandmaster.ServiceBase // inheritance

	cc *gocql.ClusterConfig
	s  *gocql.Session
}

// TODO(cmc)
func DefaultConfig(addrs ...string) *gocql.ClusterConfig {
	gocql.TimeoutLimit = 10
	cluster := gocql.NewCluster(addrs...)
	cluster.Consistency = gocql.LocalQuorum
	cluster.NumConns = int(2)
	cluster.ConnectTimeout = time.Second * 30
	cluster.Timeout = time.Second * 30

	return cluster
}

// TODO(cmc)
func New(cc *gocql.ClusterConfig) bandmaster.Service {
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(),
		cc:          cc,
	}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (s *Service) Start(
	ctx context.Context, deps map[string]bandmaster.Service,
) error {
	session, err := s.cc.CreateSession()
	if err != nil {
		return err
	}

	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		if err := session.Query(
			"SELECT cql_version FROM system.local",
		).WithContext(ctx).Exec(); err != nil {
			session.Close()
			errC <- err
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

	s.s = session
	return nil
}

// TODO(cmc)
func (s *Service) Stop(ctx context.Context) error {
	errC := make(chan error, 1)
	go func() {
		s.s.Close()
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

	return nil
}

// TODO(cmc)
func (s *Service) String() string {
	return s.ServiceBase.String() + fmt.Sprintf(" @ %v", s.cc.Hosts)
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func Client(s Service) *gocql.Session { return s.s }
