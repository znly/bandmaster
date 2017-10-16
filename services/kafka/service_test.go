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

package kafka

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/znly/bandmaster"
	"github.com/znly/bandmaster/services"
)

// -----------------------------------------------------------------------------

func TestService_Kafka(t *testing.T) {
	env, _ := NewEnv("BM_KAFKA")
	assert.NotNil(t, env)
	conf := env.Config()
	assert.NotNil(t, conf)
	conf.ConsumerTopics = []string{"test"}
	conf.ConsumerGroupID = "cg"

	s := New(conf)

	services.TestService_Generic(t, s,
		func(t *testing.T, s bandmaster.Service) {
			c := Consumer(s)
			assert.NotNil(t, c)
			assert.NotNil(t, c.Messages())
			p := Producer(s)
			assert.NotNil(t, p)
			assert.NotNil(t, p.Input())
			cc := Conf(s)
			assert.NotNil(t, cc)
			assert.Equal(t, conf, cc)
		},
	)
}

func TestService_Kafka_ProducerOnly(t *testing.T) {
	env, _ := NewEnv(uuid.New().String())
	assert.NotNil(t, env)
	conf := env.Config()
	assert.NotNil(t, conf)
	conf.ConsumerTopics = nil
	conf.ConsumerGroupID = ""

	s := New(conf)

	services.TestService_Generic(t, s,
		func(t *testing.T, s bandmaster.Service) {
			c := Consumer(s)
			assert.Nil(t, c)
			p := Producer(s)
			assert.NotNil(t, p)
			assert.NotNil(t, p.Input())
			cc := Conf(s)
			assert.NotNil(t, cc)
			assert.Equal(t, conf, cc)
		},
	)
}
