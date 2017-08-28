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
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/znly/bandmaster"
	"github.com/znly/bandmaster/services"
)

// -----------------------------------------------------------------------------

func TestService_CQL(t *testing.T) {
	env, _ := NewEnv(uuid.New().String())
	assert.NotNil(t, env)
	conf := env.Config()
	assert.NotNil(t, conf)

	// If DisableInitialHostLookup then the driver will not attempt to get
	// host info from the system.peers table, this will mean that the driver
	// will connect to hosts supplied and will not attempt to lookup the hosts
	// information, this will mean that data_centre, rack and token
	// information will not be available and as such host filtering and token
	// aware query routing will not be available.
	conf.DisableInitialHostLookup = true

	services.TestService_Generic(t, New(conf),
		func(t *testing.T, s bandmaster.Service) {
			c := Client(s)
			assert.NotNil(t, c)
			cc := Config(s)
			assert.NotNil(t, cc)
			assert.Equal(t, conf, cc)
		},
	)
}
