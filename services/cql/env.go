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
	"time"

	"github.com/gocql/gocql"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// Env can be used to configure a CQL session via the environment.
//
// It comes with sane defaults for a local development set-up.
type Env struct {
	Addrs           []string      `envconfig:"ADDRS" default:"localhost:9042"`
	ProtocolVersion int           `envconfig:"PROTOCOL_VERSION" default:"2"`
	TimeoutConnect  time.Duration `envconfig:"TIMEOUT_CONNECT" default:"30s"`
	TimeoutLimit    int64         `envconfig:"TIMEOUT_LIMIT" default:"10"`
	Timeout         time.Duration `envconfig:"TIMEOUT" default:"30s"`
	NbConns         int           `envconfig:"NB_CONNS" default:"8"`
	Consistency     Consistency   `envconfig:"CONSISTENCY" default:"LOCAL_QUORUM"`
	HostFilter      string        `envconfig:"HOST_FILTER" default:""`
}

// NewEnv parses the environment and returns a new `Env` structure.
//
// `prefix` defines the prefix for the environment keys, e.g. with a 'XX' prefix,
// 'REPLICAS' would become 'XX_REPLICAS'.
func NewEnv(prefix string) (*Env, error) {
	e := &Env{}
	if err := envconfig.Process(prefix, e); err != nil {
		return nil, errors.WithStack(err)
	}
	return e, nil
}

// Config returns a `gocql.ClusterConfig` using the values from the environment.
func (e *Env) Config() *gocql.ClusterConfig {
	gocql.TimeoutLimit = e.TimeoutLimit
	cluster := gocql.NewCluster(e.Addrs...)
	// protocol version
	cluster.ProtoVersion = e.ProtocolVersion
	// consistency level
	cluster.Consistency = gocql.Consistency(e.Consistency)
	// initial connection timeout, used during initial dial to server
	cluster.ConnectTimeout = e.TimeoutConnect
	// connection timeout
	cluster.Timeout = e.Timeout
	// number of connections per host
	cluster.NumConns = e.NbConns
	// hostfilter for multi-DC support
	if len(e.HostFilter) > 0 {
		cluster.HostFilter = gocql.DataCentreHostFilter(e.HostFilter)
	}

	return cluster
}

// -----------------------------------------------------------------------------

// Consistency is used to configure CQL's consistency value via the
// environment.
type Consistency gocql.Consistency

func (gcd *Consistency) Decode(c string) error {
	*gcd = Consistency(gocql.ParseConsistency(c))
	return nil
}
func (gcd Consistency) String() string {
	return (gocql.Consistency(gcd)).String()
}
