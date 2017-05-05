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
	"context"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

func TestService_Kafka(t *testing.T) {
	conf := DefaultConfig(128, sarama.V0_10_1_0)
	assert.NotNil(t, conf)

	s := New(conf, []string{"localhost:9092"}, []string{"test"}, "cg")
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
	c := Consumer(bs)
	assert.NotNil(t, c)
	assert.NotNil(t, c.Messages())
	p := Producer(bs)
	assert.NotNil(t, p)
	assert.NotNil(t, p.Input())

	return

	err = s.Stop(context.Background())
	assert.Nil(t, err)
	/* idempotency */
	err = s.Stop(context.Background())
	assert.Nil(t, err)

	ctx, canceller = context.WithCancel(context.Background())
	canceller()
	err = s.Stop(ctx)
	assert.Equal(t, context.Canceled, err)

	/* restart support */
	err = errors.Cause(<-m.StartAll(ctx))
	assert.Nil(t, err)
	err = <-m.StopAll(context.Background())
	assert.Nil(t, err)
}
