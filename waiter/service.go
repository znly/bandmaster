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

package waiter

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type Service struct {
	*bandmaster.ServiceBase // inheritance

	lifetime  time.Duration
	lifecyle  context.Context
	canceller context.CancelFunc
}

// TODO(cmc)
func DefaultConfig() time.Duration { return time.Second * 3 }

// TODO(cmc)
func New(lifetime time.Duration) bandmaster.Service {
	ctx, canceller := context.WithCancel(context.Background())
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(),
		lifetime:    lifetime,
		lifecyle:    ctx,
		canceller:   canceller,
	}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (s *Service) Start(
	ctx context.Context, deps map[string]bandmaster.Service,
) error {
	select { // simulate boot latency
	case <-time.After(time.Second * time.Duration(1+rand.Intn(2))):
	case <-ctx.Done():
		return ctx.Err()
	}

	go func() {
		for _, dep := range deps {
			log.Printf("hello there, %s!", dep)
		}
		<-time.After(s.lifetime) // stay alive for `lifetime`
		s.canceller()
	}()

	return nil
}

// TODO(cmc)
func (s *Service) Stop(ctx context.Context) error {
	errC := make(chan error, 1)
	go func() {
		defer close(errC)
		<-s.lifecyle.Done()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO(cmc)
func (s *Service) String() string {
	return s.ServiceBase.String() + fmt.Sprintf(" @ (%v)", s.lifetime)
}
