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

	"go.uber.org/zap"

	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
var GlobalMaestro = NewMaestro()

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
func (m *Maestro) AddService(name string, req bool, s Service, deps ...string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.services[name]; ok {
		panic(fmt.Sprintf("`%s`: service already exists", name))
	}

	base := serviceBase(s) // will panic if `s` doesn't inherit properly
	if base == nil {
		panic(fmt.Sprintf("`%s`: service *must* inherit from `ServiceBase`", name))
	}

	base.setName(name)
	base.addDependency(deps...)

	m.services[name] = s
}

// TODO(cmc)
func (m *Maestro) StartAll(ctx context.Context) <-chan error {
	m.lock.Lock()
	defer m.lock.Unlock()

	errC := make(chan error, len(m.services)+1)
	defer close(errC)

	zap.L().Debug("looking for non-existent dependencies...")
	if m.hasNonexistentDeps() {
		errC <- errors.New("non-existent dependencies") // TODO(cmc): real error
		return errC
	}
	zap.L().Debug("looking for circular dependencies...")
	if m.hasCircularDeps() {
		errC <- errors.New("circular dependencies") // TODO(cmc): real error
		return errC
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(m.services))
	for _, s := range m.services {
		go func(ss Service) {
			defer wg.Done()
			err := m.start(ctx, ss)
			if err != nil {
				errC <- err
			}
		}(s)
	}
	wg.Wait()

	return errC
}

// TODO(cmc)
// EXPECTS LOCK
func (m *Maestro) start(ctx context.Context, s Service) error {
	zap.L().Info("starting service...", zap.String("service", s.Name()))

	base := serviceBase(s)
	defer close(base.started)

	for dep := range base.Dependencies() {
		zap.L().Debug("waiting for dependency",
			zap.String("service", s.Name()), zap.String("dependency", dep),
		)
		err := <-m.services[dep].Started()
		err = errors.Wrapf(err, // TODO(cmc): real error
			"`%s`: service couldn't start because dependency `%s` failed to start",
			s.Name(), dep,
		)
		if err != nil {
			zap.L().Debug("dependency unavailable",
				zap.String("service", s.Name()), zap.String("dependency", dep),
			)
			base.started <- err
			return err
		}
		zap.L().Debug("dependency ready",
			zap.String("service", s.Name()), zap.String("dependency", dep),
		)
	}

	err := s.Start(ctx)
	err = errors.Wrapf(err, "`%s`: service failed to start", s.Name())
	if err != nil {
		zap.L().Info("service failed to start", zap.String("service", s.Name()))
		base.started <- err
		return err
	}

	zap.L().Info("service started", zap.String("service", s.Name()))

	return nil
}

// -----------------------------------------------------------------------------

// TODO(cmc)
// EXPECTS LOCK
func (m *Maestro) hasNonexistentDeps() bool {
	for _, s := range m.services {
		base := serviceBase(s)
		for dep := range base.Dependencies() {
			if _, ok := m.services[dep]; !ok {
				return true
			}
		}
	}
	return false
}

// TODO(cmc)
// EXPECTS LOCK
func (m *Maestro) hasCircularDeps() bool {
	var hasCircularDepsRec func(
		cur, parent Service, met map[string]struct{}, lvl uint,
	) bool
	hasCircularDepsRec = func(
		cur, parent Service, met map[string]struct{}, lvl uint,
	) bool {
		zap.L().Debug("checking circular dependency",
			zap.Uint("level", lvl),
			zap.String("parent", parent.Name()), zap.String("current", cur.Name()),
		)
		if _, ok := met[cur.Name()]; ok {
			return true
		}
		metRec := make(map[string]struct{}, len(met))
		for name := range met {
			metRec[name] = struct{}{}
		}
		metRec[cur.Name()] = struct{}{}
		for name := range serviceBase(cur).Dependencies() {
			if hasCircularDepsRec(m.services[name], cur, metRec, lvl+1) {
				return true
			}
		}
		return false
	}

	for _, s := range m.services {
		if hasCircularDepsRec(s, s, map[string]struct{}{}, 0) {
			return true
		}
	}

	return false
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func serviceBase(s Service) *ServiceBase {
	base := reflect.ValueOf(s).Elem().FieldByName("ServiceBase")
	if base.Kind() == reflect.Invalid {
		return nil
	}
	return base.Interface().(*ServiceBase)
}
