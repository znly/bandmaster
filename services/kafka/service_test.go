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

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/znly/bandmaster"
	"github.com/znly/bandmaster/services"
)

// -----------------------------------------------------------------------------

func TestService_Kafka(t *testing.T) {
	conf := DefaultConfig(128, sarama.V0_10_1_0)
	assert.NotNil(t, conf)
	s := New(conf, []string{"localhost:9092"}, []string{"test"}, "cg")

	services.TestService_Generic(t, s,
		func(t *testing.T, s bandmaster.Service) {
			c := Consumer(s)
			assert.NotNil(t, c)
			assert.NotNil(t, c.Messages())
			p := Producer(s)
			assert.NotNil(t, p)
			assert.NotNil(t, p.Input())
			cc := Config(s)
			assert.NotNil(t, c)
			assert.Equal(t, conf, cc)
		},
	)
}
