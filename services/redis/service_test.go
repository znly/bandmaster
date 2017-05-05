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

package redis

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

func TestService_Redis(t *testing.T) {
	conf := DefaultConfig("redis://localhost:6379/0")
	assert.NotNil(t, conf)

	s := New(conf)
	assert.NotNil(t, s)

	m := bandmaster.NewMaestro()
	m.AddService("A", true, s)

	ctx, canceller := context.WithCancel(context.Background())
	canceller()
	err := errors.Cause(<-m.StartAll(ctx))
	errExpected := &bandmaster.Error{
		Kind:       bandmaster.ErrServiceStartFailure,
		Service:    s,
		ServiceErr: context.Canceled,
	}
	assert.NotNil(t, err)
	assert.Equal(t, errExpected, err)

	err = errors.Cause(<-m.StartAll(ctx))
	assert.Nil(t, err)
	/* idempotency */
	err = errors.Cause(<-m.StartAll(ctx))
	assert.Nil(t, err)

	var bs bandmaster.Service = s
	c := Client(bs)
	assert.NotNil(t, c)
	conn := c.Get()
	assert.NotNil(t, conn)
	defer conn.Close()
	_, err = conn.Do("PING")
	assert.Nil(t, err)

	err = <-m.StopAll(context.Background())
	assert.Nil(t, err)
	/* idempotency */
	err = <-m.StopAll(context.Background())
	assert.Nil(t, err)

	ctx, canceller = context.WithCancel(context.Background())
	canceller()
	err = errors.Cause(<-m.StopAll(context.Background()))
	errExpected = &bandmaster.Error{
		Kind:       bandmaster.ErrServiceStopFailure,
		Service:    s,
		ServiceErr: context.Canceled,
	}
	assert.NotNil(t, err)
	assert.Equal(t, errExpected, err)

	/* restart support */
	err = <-m.StartAll(ctx)
	assert.Nil(t, err)
	err = <-m.StopAll(context.Background())
	assert.Nil(t, err)
}
