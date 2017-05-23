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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/fatih/camelcase"
	"github.com/gocql/gocql"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// Env can be used to configure a CQL session via the environment.
//
// It comes with sane default for a local development set-up.
type Env struct {
	Consistency    Consistency   `envconfig:"CONSISTENCY" default:"LOCAL_QUORUM"`
	ConnectTimeout time.Duration `envconfig:"CONNECT_TIMEOUT" default:"45s"`
	Timeout        time.Duration `envconfig:"TIMEOUT" default:"30s"`
	NbConns        uint          `envconfig:"NB_CONNS" default:"2"`
	Addrs          []string      `envconfig:"ADDRS" default:"localhost:9042"`

	TimeoutLimit uint `envconfig:"TIMEOUT_LIMIT" default:"10"`
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
	cluster := gocql.NewCluster(e.Addrs...)
	// default consistency level
	cluster.Consistency = gocql.Consistency(e.Consistency)
	// initial connection timeout, used during initial dial to server
	cluster.ConnectTimeout = e.ConnectTimeout
	// connection timeout
	cluster.Timeout = e.Timeout
	// number of connections per host
	cluster.NumConns = int(e.NbConns)

	return cluster
}

// -----------------------------------------------------------------------------

func (e *Env) String() string {
	cv := reflect.ValueOf(*e)
	fields := reflect.TypeOf(*e)
	fieldStrs := make([]string, fields.NumField())
	for i := 0; i < fields.NumField(); i++ {
		f := fields.Field(i)
		if len(f.Name) > 0 && strings.ToLower(f.Name[:1]) == f.Name[:1] {
			continue // private field
		}
		fNameParts := camelcase.Split(f.Name)
		for i, fnp := range fNameParts {
			fNameParts[i] = strings.ToUpper(fnp)
		}
		fName := strings.Join(fNameParts, "_")
		itf := cv.Field(i).Interface()
		if _, ok := itf.(fmt.Stringer); ok {
			fieldStrs[i] = fmt.Sprintf("%s = %s", fName, itf)
		} else {
			fieldStrs[i] = fmt.Sprintf("%s = %v", fName, itf)
		}
	}
	return strings.Join(fieldStrs, "\n") + "\n"
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
