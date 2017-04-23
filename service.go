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
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type Service interface {
	Start(ctx context.Context) error

	Name() string
	Required() bool

	Started() <-chan error
	Stopped() <-chan error
}

// -----------------------------------------------------------------------------

// TODO(cmc)
type ServiceBase struct {
	lock *sync.RWMutex

	name       string
	required   bool
	directDeps map[string]struct{}

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

// -----------------------------------------------------------------------------

// TODO(cmc)
func (sb *ServiceBase) addDependency(deps ...string) {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	for _, dep := range deps {
		if dep == sb.name {
			panic(fmt.Sprintf("`%s`: service depends on itself", sb.name))
		}
		if _, ok := sb.directDeps[dep]; ok {
			panic(fmt.Sprintf(
				"`%s`: service already depends on `%s`", sb.name, dep,
			))
		}
		sb.directDeps[dep] = struct{}{}
	}
}

// TODO(cmc)
func (sb *ServiceBase) Dependencies() map[string]struct{} {
	sb.lock.RLock()
	defer sb.lock.RUnlock()

	deps := make(map[string]struct{}, len(sb.directDeps))
	for dep := range sb.directDeps {
		deps[dep] = struct{}{}
	}

	return deps
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (sb *ServiceBase) Started() <-chan error { return sb.started }
func (sb *ServiceBase) Stopped() <-chan error { return sb.stopped }
