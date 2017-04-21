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
	"reflect"
	"sync"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type Service interface {
	Run(bootCtx, lifeCtx context.Context) (<-chan error, <-chan error)

	Name() string
	Started() <-chan struct{}
	Stopped() <-chan struct{}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
type ServiceDependency struct {
	name          string
	dontWaitForMe bool
}

// TODO(cmc)
func NewServiceDependency(name string, dontWaitForMe bool) ServiceDependency {
	return ServiceDependency{name: name, dontWaitForMe: dontWaitForMe}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
type ServiceBase struct {
	lock       *sync.RWMutex
	name       string
	directDeps map[string]ServiceDependency

	started chan struct{}
	stopped chan struct{}
}

// TODO(cmc)
func NewServiceBase() *ServiceBase {
	return &ServiceBase{lock: &sync.RWMutex{}}
}

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
func (sb *ServiceBase) addDependency(deps ...ServiceDependency) {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	for _, dd := range sb.directDeps {
		if dd.name == sb.name {
			panic(fmt.Sprintf("`%s`: service depends on itself", sb.name))
		}
		if _, ok := sb.directDeps[dd.name]; ok {
			panic(fmt.Sprintf(
				"`%s`: service already depends on `%s`", sb.name, dd.name,
			))
		}
		sb.directDeps[dd.name] = dd
	}
}
func (sb *ServiceBase) Dependencies() map[string]ServiceDependency {
	sb.lock.RLock()
	defer sb.lock.RUnlock()

	deps := make(map[string]ServiceDependency, len(sb.directDeps))
	for name, dd := range sb.directDeps {
		deps[name] = dd
	}

	return deps
}

// TODO(cmc)
func (sb *ServiceBase) Started() <-chan struct{} { return sb.started }
func (sb *ServiceBase) Stopped() <-chan struct{} { return sb.stopped }

// -----------------------------------------------------------------------------

// TODO(cmc)
func serviceBase(s Service) *ServiceBase {
	base := reflect.ValueOf(s).Elem().FieldByName("ServiceBase").Interface()
	return base.(*ServiceBase)
}
