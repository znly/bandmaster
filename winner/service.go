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

package winner

import (
	"context"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type Service struct {
	*bandmaster.ServiceBase // inheritance

	lifetime time.Duration
}

// TODO(cmc)
func New(lifetime time.Duration) bandmaster.Service {
	return &Service{ServiceBase: bandmaster.NewServiceBase(), lifetime: lifetime}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (s *Service) Run(
	bootCtx, lifeCtx context.Context,
) (<-chan error, <-chan error) {
	bootErrC := make(chan error, 1)
	lifeErrC := make(chan error, 1)

	go func() {
		defer close(lifeErrC) // stopped

		select { // simulate boot latency
		case <-time.After(time.Second * time.Duration(1+rand.Intn(2))):
		case <-bootCtx.Done():
			bootErrC <- errors.WithStack(bootCtx.Err())
			return
		}
		close(bootErrC) // started

		select {
		case <-time.After(s.lifetime):
		case <-lifeCtx.Done():
		}
	}()

	return bootErrC, lifeErrC
}