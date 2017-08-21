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
	"fmt"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type Service interface {
	Start(ctx context.Context, deps map[string]Service) error
	Stop(ctx context.Context) error

	Name() string
	Required() bool
	RetryConf() (uint, time.Duration)

	String() string

	Started(ctx context.Context) <-chan error
	Stopped(ctx context.Context) <-chan error
}

// -----------------------------------------------------------------------------

// TODO(cmc)
type ServiceBase struct {
	lock *sync.RWMutex

	name           string
	required       bool
	retries        uint
	initialBackoff time.Duration
	directDeps     map[string]struct{}

	started chan error
	stopped chan error
}

// TODO(cmc)
func NewServiceBase() *ServiceBase {
	return &ServiceBase{
		lock:       &sync.RWMutex{},
		directDeps: map[string]struct{}{},
		started:    make(chan error, 1),
		stopped:    make(chan error, 1),
	}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (sb *ServiceBase) setName(name string) {
	sb.lock.Lock()
	sb.name = name
	sb.lock.Unlock()
}
func (sb *ServiceBase) Name() string {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	return sb.name
}

// TODO(cmc)
func (sb *ServiceBase) setRequired(required bool) {
	sb.lock.Lock()
	sb.required = required
	sb.lock.Unlock()
}
func (sb *ServiceBase) Required() bool {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	return sb.required
}

// TODO(cmc)
func (sb *ServiceBase) setRetryConf(retries uint, initialBackoff time.Duration) {
	sb.lock.Lock()
	sb.retries = retries
	sb.initialBackoff = initialBackoff
	sb.lock.Unlock()
}
func (sb *ServiceBase) RetryConf() (uint, time.Duration) {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	return sb.retries, sb.initialBackoff
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (sb *ServiceBase) String() string {
	name := sb.Name()
	req := "optional"
	if sb.Required() {
		req = "required"
	}
	return fmt.Sprintf("'%s' [%s]", name, req)
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (sb *ServiceBase) addDependency(deps ...string) {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	for _, dep := range deps {
		if dep == sb.name {
			panic(&Error{Kind: ErrServiceDependsOnItself, ServiceName: sb.name})
		}
		if _, ok := sb.directDeps[dep]; ok {
			panic(&Error{
				Kind:        ErrServiceDuplicateDependency,
				ServiceName: sb.name,
				Dependency:  dep,
			})
		}
		sb.directDeps[dep] = struct{}{}
	}
}

// TODO(cmc)
func (sb *ServiceBase) Dependencies() map[string]struct{} {
	sb.lock.RLock()
	defer sb.lock.RUnlock()

	deps := make(map[string]struct{}, len(sb.directDeps)) // copy
	for dep := range sb.directDeps {
		deps[dep] = struct{}{}
	}

	return deps
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (sb *ServiceBase) Started(ctx context.Context) <-chan error {
	return cloneErrChannel(ctx, sb.started)
}

// TODO(cmc)
func (sb *ServiceBase) Stopped(ctx context.Context) <-chan error {
	return cloneErrChannel(ctx, sb.stopped)
}

// TODO(cmc)
func cloneErrChannel(ctx context.Context, c chan error) <-chan error {
	errC := make(chan error, cap(c))
	go func() {
		defer close(errC)
		select {
		case err := <-c:
			c <- err
			errC <- err
		case <-ctx.Done():
			errC <- ctx.Err()
		}
	}()
	return errC
}