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

package bandmaster

import (
	"context"
	"errors"
	"sync"
)

// -----------------------------------------------------------------------------

var _testServiceErrNotAllowedToStart = errors.New("not allowed to start yet!")

// TestServiceimplements a Service for testing purposes.
type TestService struct {
	*ServiceBase // "inheritance"

	lock             *sync.RWMutex
	dontStart        bool // retry backoff
	started, stopped bool
}

func NewTestService() *TestService {
	return &TestService{ServiceBase: NewServiceBase(), lock: &sync.RWMutex{}}
}

func (s *TestService) Start(context.Context, map[string]Service) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.dontStart {
		return _testServiceErrNotAllowedToStart
	}
	s.started = true

	return nil
}

func (s *TestService) Stop(context.Context) error {
	s.stopped = true
	return nil // always succeed
}
