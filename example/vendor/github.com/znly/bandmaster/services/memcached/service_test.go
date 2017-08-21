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

package memcached

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/znly/bandmaster"
	"github.com/znly/bandmaster/services"
)

// -----------------------------------------------------------------------------

func TestService_Memcached(t *testing.T) {
	services.TestService_Generic(t, New(DefaultConfig("localhost:11211")),
		func(t *testing.T, s bandmaster.Service) {
			c := Client(s)
			assert.NotNil(t, c)
		},
	)
}
