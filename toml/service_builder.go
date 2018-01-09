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

package toml

import (
	"github.com/pkg/errors"
	"github.com/znly/bandmaster"
	bm_cql "github.com/znly/bandmaster/services/cql"
	bm_es1 "github.com/znly/bandmaster/services/es1"
	bm_es2 "github.com/znly/bandmaster/services/es2"
	bm_es5 "github.com/znly/bandmaster/services/es5"
	bm_kafka "github.com/znly/bandmaster/services/kafka"
	bm_memcached "github.com/znly/bandmaster/services/memcached"
	bm_nats "github.com/znly/bandmaster/services/nats"
	bm_redis "github.com/znly/bandmaster/services/redis"
)

// -----------------------------------------------------------------------------

// A ServiceBuilder is a user-defined function that builds a Service based on a
// given type, using the specified name to retrieve its configuration from the
// environment.
// A ServiceBuilder is passed to the LoadAsEnv function in order to build the
// Services extracted from a TOML file.
//
// This package embeds a default ServiceBuilder that is capable of building
// every type of services supported by BandMaster, and is always passed to
// LoadAsEnv no matter what.
type ServiceBuilder func(typ string, name string) (bandmaster.Service, error)

// -----------------------------------------------------------------------------

// TODO(cmc): do better.

type serviceType = string

const (
	kafka     serviceType = "kafka"
	cql       serviceType = "cql"
	es1       serviceType = "es1"
	es2       serviceType = "es2"
	es5       serviceType = "es5"
	memcached serviceType = "memcached"
	nats      serviceType = "nats"
	redis     serviceType = "redis"
)

// NOTE(cmc): this has to import every provided services' packages.
// NOTE(cmc): also this is way ugly.
var defaultBuilder = func(typ string, name string) (bandmaster.Service, error) {
	switch typ {
	case kafka:
		env, err := bm_kafka.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_kafka.New(env.Config()), nil
	case cql:
		env, err := bm_cql.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_cql.New(env.Config()), nil
	case es1:
		env, err := bm_es1.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_es1.New(env.Config()), nil
	case es2:
		env, err := bm_es2.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_es2.New(env.Config()), nil
	case es5:
		env, err := bm_es5.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_es5.New(env.Config()), nil
	case memcached:
		env, err := bm_memcached.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_memcached.New(env.Config()), nil
	case nats:
		env, err := bm_nats.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_nats.New(env.Config()), nil
	case redis:
		env, err := bm_redis.NewEnv(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return bm_redis.New(env.Config()), nil
	}
	return nil, nil
}
