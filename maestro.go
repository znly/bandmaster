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
	"fmt"
	"sync"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type Maestro struct {
	lock     *sync.RWMutex
	services map[string]Service
}

// TODO(cmc)
func NewMaestro() *Maestro {
	return &Maestro{lock: &sync.RWMutex{}, services: map[string]Service{}}
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (m *Maestro) AddService(name string, s Service, deps ...ServiceDependency) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.services[name]; ok {
		panic(fmt.Sprintf("`%s`: service already exists", name))
	}

	base := serviceBase(s) // will panic if `s` doesn't inherit properly
	base.setName(name)
	base.addDependency(deps...)

	m.services[name] = s
}

// TODO(cmc)
func (m *Maestro) StartAll(ctx context.Context) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.hasCircularDeps() {
		return errors.New("circular deps") // TODO(cmc): real error
	}

	//for name, s range m.services {
	//}

	return nil
}

// TODO(cmc)
// EXPECTS LOCK
//func (m *Maestro) start(ctx context.Context, s Service) {
//}

// -----------------------------------------------------------------------------

// TODO(cmc)
// EXPECTS LOCK
func (m *Maestro) hasCircularDeps() bool {
	var hasCircularDepsRec func(s Service, met map[string]struct{}) bool
	hasCircularDepsRec = func(s Service, met map[string]struct{}) bool {
		if _, ok := met[s.Name()]; ok {
			return true
		}
		met[s.Name()] = struct{}{}
		for name := range serviceBase(s).Dependencies() {
			if hasCircularDepsRec(m.services[name], met) {
				return true
			}
		}
		return false
	}

	for _, s := range m.services {
		if hasCircularDepsRec(s, map[string]struct{}{}) {
			return true
		}
	}

	return false
}
