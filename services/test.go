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

package services

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
func TestService_Generic(t *testing.T, s bandmaster.Service,
	specifics func(t *testing.T, s bandmaster.Service),
) {
	assert.NotNil(t, s)

	m := bandmaster.NewMaestro()
	m.AddServiceWithBackoff("A", true, 10, time.Millisecond*200, s)

	ctx, canceller := context.WithCancel(context.Background())
	canceller()
	errExpected := &bandmaster.Error{
		Kind:       bandmaster.ErrServiceStartFailure,
		Service:    s,
		ServiceErr: context.Canceled,
	}
	err := errors.Cause(<-m.StartAll(ctx))
	assert.Error(t, err)
	assert.Equal(t, errExpected, err)
	/* idempotency (error) */
	err = errors.Cause(<-m.StartAll(ctx))
	assert.Error(t, err)
	assert.Equal(t, errExpected, err)

	err = errors.Cause(<-m.StartAll(context.Background()))
	assert.NoError(t, err)
	/* idempotency (success) */
	err = errors.Cause(<-m.StartAll(context.Background()))
	assert.NoError(t, err)

	if specifics != nil {
		specifics(t, s)
	}

	ctx, canceller = context.WithCancel(context.Background())
	canceller()
	errExpected = &bandmaster.Error{
		Kind:       bandmaster.ErrServiceStopFailure,
		Service:    s,
		ServiceErr: context.Canceled,
	}
	err = errors.Cause(<-m.StopAll(ctx))
	assert.Error(t, err)
	assert.Equal(t, errExpected, err)
	/* idempotency (error) */
	err = errors.Cause(<-m.StopAll(ctx))
	assert.Error(t, err)
	assert.Equal(t, errExpected, err)

	err = <-m.StopAll(context.Background())
	assert.NoError(t, err)
	/* idempotency (success) */
	err = <-m.StopAll(context.Background())
	assert.NoError(t, err)

	/* restart support */
	err = <-m.StartAll(context.Background())
	assert.NoError(t, err)
	if specifics != nil {
		specifics(t, s)
		return
	}
	err = <-m.StopAll(context.Background())
	assert.NoError(t, err)
}
