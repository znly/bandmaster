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
	"reflect"
	"sync"

	"go.uber.org/zap"

	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
var (
	_gm     = NewMaestro()
	_gmLock = &sync.RWMutex{}
)

// TODO(cmc)
func GlobalMaestro() *Maestro {
	_gmLock.RLock()
	defer _gmLock.RUnlock()

	return _gm
}

// TODO(cmc)
func ReplaceGlobalMaestro(m *Maestro) *Maestro {
	_gmLock.Lock()
	cur := _gm
	_gm = m
	_gmLock.Unlock()
	return cur
}

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
func (m *Maestro) AddService(name string, req bool, s Service, deps ...string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.services[name]; ok {
		panic(&Error{kind: ErrServiceAlreadyExists, serviceName: name})
	}

	base := serviceBase(s)
	if base == nil { // panic if `s` doesn't inherit properly
		panic(&Error{kind: ErrServiceWithoutBase, serviceName: name})
	}

	base.setName(name)
	base.setRequired(req)
	base.addDependency(deps...)

	m.services[name] = s
}

// TODO(cmc)
func (m *Maestro) Service(name string) Service {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.services[name]
}

// -----------------------------------------------------------------------------

// TODO(cmc)
// Nothing is started in case of missing or circular dependencies.
func (m *Maestro) StartAll(ctx context.Context) <-chan error {
	m.lock.Lock()
	defer m.lock.Unlock()

	errC := make(chan error, len(m.services)+1)
	defer close(errC)

	zap.L().Debug("looking for missing dependencies...")
	if m.hasMissingDeps(errC) {
		return errC
	}
	zap.L().Debug("looking for circular dependencies...")
	if m.hasCircularDeps(errC) {
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
	name := s.Name()
	zap.L().Info("starting service...", zap.String("service", name))

	base := serviceBase(s)
	defer func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok &&
					err.Error() == "close of closed channel" {
					return // ignore panic, start() is idempotent
				}
				panic(r) // not a CCC error, forward panic
			}
		}()
		defer close(base.started)
	}()

	deps := make(map[string]Service, len(base.Dependencies()))
	for dep := range base.Dependencies() {
		zap.L().Debug("waiting for dependency to start",
			zap.String("service", name), zap.String("dependency", dep),
		)
		d := m.services[dep]
		err := <-d.Started()
		if err != nil {
			zap.L().Debug("dependency failed to start",
				zap.String("service", name), zap.String("dependency", dep),
			)
			err = errors.WithStack(&Error{
				service:    s,
				dependency: dep,
				kind:       ErrDependencyUnavailable,
			})
			base.started <- err
			return err
		}
		deps[dep] = d
		zap.L().Debug("dependency ready",
			zap.String("service", name), zap.String("dependency", dep),
		)
	}

	err := s.Start(ctx, deps)
	if err != nil {
		zap.L().Info("service failed to start",
			zap.String("service", name), zap.String("err", err.Error()),
		)
		err = &Error{
			kind:       ErrServiceStartFailure,
			service:    s,
			serviceErr: err,
		}
		base.started <- err
		return err
	}

	zap.L().Info("service successfully started", zap.String("service", s.String()))

	return nil
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (m *Maestro) StopAll(ctx context.Context) <-chan error {
	m.lock.Lock()
	defer m.lock.Unlock()

	errC := make(chan error, len(m.services)+1)
	defer close(errC)

	wg := &sync.WaitGroup{}
	wg.Add(len(m.services))
	for _, s := range m.services {
		go func(ss Service) {
			defer wg.Done()
			err := m.stop(ctx, ss)
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
func (m *Maestro) stop(ctx context.Context, s Service) error {
	name := s.Name()
	zap.L().Info("stopping service...", zap.String("service", name))

	base := serviceBase(s)
	defer func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok &&
					err.Error() == "close of closed channel" {
					return // ignore panic, stop() is idempotent
				}
				panic(r) // not a CCC error, forward panic
			}
		}()
		defer close(base.stopped)
	}()

	for dep := range base.Dependencies() {
		zap.L().Debug("waiting for dependency to stop",
			zap.String("service", name), zap.String("dependency", dep),
		)
		err := <-m.services[dep].Stopped()
		if err != nil {
			zap.L().Debug("dependency failed to stop (ignored)",
				zap.String("service", name), zap.String("dependency", dep),
			)
		} else {
			zap.L().Debug("dependency stopped",
				zap.String("service", name), zap.String("dependency", dep),
			)
		}
	}

	err := s.Stop(ctx)
	if err != nil {
		zap.L().Info("service failed to stop",
			zap.String("service", name), zap.String("err", err.Error()),
		)
		err = &Error{
			kind:       ErrServiceStopFailure,
			service:    s,
			serviceErr: err,
		}
		base.stopped <- err
		return err
	}

	zap.L().Info("service successfully stopped", zap.String("service", s.String()))

	return nil
}

// -----------------------------------------------------------------------------

// TODO(cmc)
// EXPECTS LOCK
func (m *Maestro) hasMissingDeps(errC chan error) (failure bool) {
	for _, s := range m.services {
		base := serviceBase(s)
		for dep := range base.Dependencies() {
			if _, ok := m.services[dep]; !ok {
				errC <- errors.WithStack(&Error{
					service:    s,
					dependency: dep,
					kind:       ErrDependencyMissing,
				})
				failure = true
			}
		}
	}
	return failure
}

// TODO(cmc)
// EXPECTS LOCK
func (m *Maestro) hasCircularDeps(errC chan error) bool {
	var hasCircularDepsRec func(
		cur, parent Service, met map[string]uint, lvl uint,
	) bool
	hasCircularDepsRec = func(
		cur, parent Service, met map[string]uint, lvl uint,
	) bool {
		if lvl > 0 {
			zap.L().Debug("checking circular dependencies",
				zap.Uint("level", lvl),
				zap.String("current", cur.Name()),
				zap.String("parent", parent.Name()),
			)
		}
		if _, ok := met[cur.Name()]; ok {
			circularDeps := make([]string, len(met)+1)
			for dep, lvl := range met {
				circularDeps[lvl] = dep
			}
			circularDeps[lvl] = cur.Name()
			errC <- errors.WithStack(&Error{
				service:      m.services[circularDeps[0]],
				circularDeps: circularDeps,
				kind:         ErrDependencyCircular,
			})
			return true
		}
		metRec := make(map[string]uint, len(met))
		for name, lvl := range met {
			metRec[name] = lvl
		}
		metRec[cur.Name()] = lvl
		for name := range serviceBase(cur).Dependencies() {
			if hasCircularDepsRec(m.services[name], cur, metRec, lvl+1) {
				return true
			}
		}
		return false
	}

	for _, s := range m.services {
		if hasCircularDepsRec(s, s, map[string]uint{}, 0) {
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
