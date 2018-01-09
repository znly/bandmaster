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
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/kelseyhightower/envconfig"
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

// TODO(cmc): improve docs here.
// TODO(cmc): tests.

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

// LoadAsEnv parses a TOML file that follows BandMaster's conventions and
// augments the current environment with the values found in the file, unless
// they are already defined.
//
// TOML files can be used to specify both configuration parameters (such as
// listening address, number of connections, etc..) as well as
// BandMaster-specific service-properties (see ServiceProperties).
//
// Once a file has been successfully loaded and the environment augmented with
// the new variables, the corresponding Services will be instanciated using
// either the given ServiceBuilder or the default one (provided by this package).
// If everything goes smoothly, the newly-created services are returned and can
// be added to a Maestro using the standard APIsl; otherwise, an error is
// returned.
//
// An example is the best way to get an idea of how this works:
//
//   [[ services.memcached.mc_0 ]]
//
//   [[ services.memcached.mc_1 ]]
//   	[[ services.memcached.mc_1.env ]]
//   		addr = "localhost:11211"
//   		timeout_rw = "2s"
//   	[[ services.memcached.mc_1.properties ]]
//   		required = "true"
//
//   [[ services.kafka.kafka_1 ]]
//   	[[ services.kafka.kafka_1.env ]]
//   		addrs = "localhost:9092"
//   		cons_topics = "my_tropico"
//   	[[ services.kafka.kafka_1.properties ]]
//   		required = "false"
//   		dependencies = "memcached_1"
//   		backoff = "true"
//   		backoff_max_retries = "6"
//   		backoff_initial_duration = "250ms"
//
// Note that using this API will force you to import every services/
// subpackages.
func LoadAsEnv(
	path string,
	builder ServiceBuilder,
) (map[string]bandmaster.Service, error) {
	final := map[string]bandmaster.Service{}

	conf := struct {
		Services map[string]map[string][]struct {
			Env        []map[string]string `toml:"env"`
			Properties []map[string]string `toml:"properties"`
		} `toml:"services"`
	}{}
	_, err := toml.DecodeFile(path, &conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for serviceType, services := range conf.Services {
		for name, categories := range services {
			for _, attributes := range categories {
				for _, env := range attributes.Env {
					for k, v := range env {
						k = strings.ToUpper(name) + "_" + strings.ToUpper(k)
						if preset := os.Getenv(k); len(preset) <= 0 {
							if err := os.Setenv(k, v); err != nil {
								return nil, errors.WithStack(err)
							}
						}
					}
				}
				for _, properties := range attributes.Properties {
					for k, v := range properties {
						k = strings.ToUpper(name) + "_PROP" +
							"_" + strings.ToUpper(k)
						if preset := os.Getenv(k); len(preset) <= 0 {
							if err := os.Setenv(k, v); err != nil {
								return nil, errors.WithStack(err)
							}
						}
					}
				}
			}

			props := bandmaster.ServiceProperties{}
			if err := envconfig.Process(name, &props); err != nil {
				return nil, errors.WithStack(err)
			}
			var s bandmaster.Service
			if builder != nil {
				s, err = builder(serviceType, name)
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
			if s == nil {
				s, err = defaultBuilder(serviceType, name)
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
			if s == nil {
				return nil, errors.New(
					"service cannot be built using the specified ServiceBuilder")
			}

			final[name] = s
		}
	}

	return final, nil
}

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
