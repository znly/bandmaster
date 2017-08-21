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
	"time"

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
	m.AddServiceWithBackoff(name, req, 0, 0, s, deps...)
}

// TODO(cmc)
func (m *Maestro) AddServiceWithBackoff(
	name string, req bool,
	retries uint, initialBackoff time.Duration,
	s Service, deps ...string,
) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.services[name]; ok {
		panic(&Error{Kind: ErrServiceAlreadyExists, ServiceName: name})
	}

	base := serviceBase(s)
	// This method is the only legal path one can take in order to add a service
	// to the Maestro, we thus consider this check to be always true further down
	// chain.
	if base == nil { // panic if `s` doesn't inherit properly
		panic(&Error{Kind: ErrServiceWithoutBase, ServiceName: name})
	}

	base.setName(name)
	base.setRequired(req)
	base.setRetryConf(retries, initialBackoff)
	base.addDependency(deps...)

	m.services[name] = s
}

// TODO(cmc)
func (m *Maestro) Service(name string) Service {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.services[name]
}

// TODO(cmc)
func (m *Maestro) ServiceReady(ctx context.Context, name string) Service {
	m.lock.RLock()
	s := m.services[name]
	m.lock.RUnlock()

	if s == nil {
		return nil
	}
	if err := <-s.Started(ctx); err != nil {
		return nil
	}

	return s
}

// -----------------------------------------------------------------------------

// TODO(cmc)
// Nothing is started in case of missing or circular dependencies.
func (m *Maestro) StartAll(ctx context.Context) <-chan error {
	m.lock.Lock()
	defer m.lock.Unlock()

	errC := make(chan error, len(m.services)+1) // +1 because I don't like surprises
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
		// NOTE: The Maestro's lock is kept during execution of these routines.
		go func(ss Service) { // services are started in parallel
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
	base := serviceBase(s)

	/* -- Handling restarts & idempotency, canceled contexts, etc... -- */
	select {
	case err := <-base.started: // raw access to started state
		if err == nil {
			base.started <- err
			return nil // idempotency
		}
	case <-ctx.Done():
		return &Error{
			Kind:       ErrServiceStartFailure,
			Service:    s,
			ServiceErr: ctx.Err(),
		}
	default: // go on
	}

	name := s.Name()
	zap.L().Info("starting service...", zap.String("service", name))

	/* -- Wait for `s`' dependencies to be ready -- */
	deps := make(map[string]Service, len(base.Dependencies()))
	for dep := range base.Dependencies() {
		zap.L().Debug("waiting for dependency to start",
			zap.String("service", name), zap.String("dependency", dep),
		)
		d := m.services[dep]
		err := <-d.Started(ctx)
		if err != nil {
			zap.L().Debug("dependency failed to start",
				zap.String("service", name), zap.String("dependency", dep),
			)
			err = errors.WithStack(&Error{
				Kind:       ErrDependencyUnavailable,
				Service:    s,
				Dependency: dep,
			})
			base.started <- err
			return err
		}
		deps[dep] = d
		zap.L().Debug("dependency ready",
			zap.String("service", name), zap.String("dependency", dep),
		)
	}

	/* -- Start the actual service `s` -- */
	base.lock.Lock()
	var err error
	errC := make(chan error, 1)
	go func() { errC <- s.Start(ctx, deps); close(errC) }()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errC:
	}
	base.lock.Unlock()
	retries, ib := base.RetryConf()
	attempts := uint(0)
	for err != nil {
		attempts++
		if attempts >= retries {
			zap.L().Warn("service failed to start",
				zap.String("service", name), zap.String("err", err.Error()),
				zap.Uint("attempt", attempts),
			)
			err = &Error{
				Kind:       ErrServiceStartFailure,
				Service:    s,
				ServiceErr: err,
			}
			base.started <- err
			return err
		} else {
			msg := fmt.Sprintf("service failed to start, retrying in %v...", ib)
			zap.L().Info(msg,
				zap.String("service", name), zap.String("err", err.Error()),
				zap.Uint("attempt", attempts),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(ib):
				ib *= 2
			}
		}
		base.lock.Lock()
		errC := make(chan error, 1)
		go func() { errC <- s.Start(ctx, deps); close(errC) }()
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case err = <-errC:
		}
		base.lock.Unlock()
	}
	base.started <- nil // don't close this channel, ever
	select {            // successful start -> flush stopped state
	case <-base.stopped:
	default:
	}

	zap.L().Info("service successfully started", zap.String("service", s.String()))

	return nil
}

// -----------------------------------------------------------------------------

// TODO(cmc)
func (m *Maestro) StopAll(ctx context.Context) <-chan error {
	m.lock.Lock()
	defer m.lock.Unlock()

	errC := make(chan error, len(m.services)+1) // +1 because I don't like surprises
	defer close(errC)

	wg := &sync.WaitGroup{}
	wg.Add(len(m.services))
	for _, s := range m.services {
		// NOTE: The Maestro's lock is kept during execution of these routines.
		go func(ss Service) { // services are stopped in parallel
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
	base := serviceBase(s)

	/* -- Handling restarts & idempotency, canceled contexts, etc... -- */
	select {
	case err := <-base.stopped: // raw access to stopped state
		if err == nil {
			base.stopped <- err
			return nil // idempotency
		}
	case <-ctx.Done():
		return &Error{
			Kind:       ErrServiceStopFailure,
			Service:    s,
			ServiceErr: ctx.Err(),
		}
	default: // go on
	}

	name := s.Name()
	zap.L().Info("stopping service...", zap.String("service", name))

	for dep := range base.Dependencies() {
		zap.L().Debug("waiting for dependency to stop",
			zap.String("service", name), zap.String("dependency", dep),
		)
		err := <-m.services[dep].Stopped(ctx)
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

	base.lock.Lock()
	var err error
	errC := make(chan error, 1)
	go func() { errC <- s.Stop(ctx); close(errC) }()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errC:
	}
	base.lock.Unlock()
	if err != nil {
		zap.L().Info("service failed to stop",
			zap.String("service", name), zap.String("err", err.Error()),
		)
		err = &Error{
			Kind:       ErrServiceStopFailure,
			Service:    s,
			ServiceErr: err,
		}
		base.stopped <- err
		return err
	}
	base.stopped <- nil // don't close this channel, ever
	select {            // successful stop -> flush started state
	case <-base.started:
	default:
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
					Service:    s,
					Dependency: dep,
					Kind:       ErrDependencyMissing,
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
				Service:      m.services[circularDeps[0]],
				CircularDeps: circularDeps,
				Kind:         ErrDependencyCircular,
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
