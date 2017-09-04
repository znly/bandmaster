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

// Service is the main interface behind Bandmaster, any service that wishes to
// be operated by a Maestro must implement it.
//
// To ease the integration of new services into the system, a ServiceBase class
// that you can pseudo-inherit from (i.e. embed) is provided and will
// automagically fill in most of the boilerplate required to implement the
// Service interface.
// See any service implementation in "services/" folder for examples of this.
type Service interface {
	Start(ctx context.Context, deps map[string]Service) error
	Stop(ctx context.Context) error

	Name() string
	Required() bool
	RetryConf() (uint, time.Duration)

	String() string

	Started() <-chan error
	Stopped() <-chan error
}

// -----------------------------------------------------------------------------

// A ServiceBase implements most of the boilerplate required to satisfy the
// Service interface.
// You can, and should, embed it in your service structure to ease its
// integration in BandMaster.
// See any service implementation in "services/" folder for examples of this.
type ServiceBase struct {
	lock *sync.RWMutex

	name           string
	required       bool
	retries        uint
	initialBackoff time.Duration
	directDeps     map[string]struct{}

	startedCs  []chan error
	started    bool
	startedErr error
	stoppedCs  []chan error
	stopped    bool
	stoppedErr error
}

// NewServiceBase returns a properly initialized ServiceBase that you can
// embed in your service definition.
func NewServiceBase() *ServiceBase {
	return &ServiceBase{
		lock:       &sync.RWMutex{},
		directDeps: map[string]struct{}{},
		startedCs:  make([]chan error, 0, 16),
		started:    false,
		startedErr: nil,
		stoppedCs:  make([]chan error, 0, 16),
		stopped:    true,
		stoppedErr: nil,
	}
}

// -----------------------------------------------------------------------------

func (sb *ServiceBase) setName(name string) {
	sb.lock.Lock()
	sb.name = name
	sb.lock.Unlock()
}

// Name returns the name of the service; it is thread-safe.
func (sb *ServiceBase) Name() string {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	return sb.name
}

func (sb *ServiceBase) setRequired(required bool) {
	sb.lock.Lock()
	sb.required = required
	sb.lock.Unlock()
}

// Required returns true if the service is marked as required; it is
// thread-safe.
func (sb *ServiceBase) Required() bool {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	return sb.required
}

func (sb *ServiceBase) setRetryConf(retries uint, initialBackoff time.Duration) {
	sb.lock.Lock()
	sb.retries = retries
	sb.initialBackoff = initialBackoff
	sb.lock.Unlock()
}

// RetryConf returns the number of retries and the initial value used by the
// the service for exponential backoff; it is thread-safe.
func (sb *ServiceBase) RetryConf() (uint, time.Duration) {
	sb.lock.RLock()
	defer sb.lock.RUnlock()
	return sb.retries, sb.initialBackoff
}

// -----------------------------------------------------------------------------

func (sb *ServiceBase) String() string {
	name := sb.Name()
	req := "optional"
	if sb.Required() {
		req = "required"
	}
	return fmt.Sprintf("'%s' [%s]", name, req)
}

// -----------------------------------------------------------------------------

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

// Dependencies returns a set of the direct dependencies of the service.
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

func (sb *ServiceBase) start(err error) error {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	/* one or more of our dependencies have failed to start */
	if err != nil {
		sb.started = false
		sb.startedErr = err
		sb.stopped = !sb.started
		sb.stoppedErr = nil
		for _, startedC := range sb.startedCs {
			startedC <- err
		}
		return err
	}

	/* we're already running, this is a noop */
	if sb.started {
		return nil
	}

	/* successful boot */
	sb.started = true
	sb.startedErr = nil
	sb.stopped = !sb.started
	sb.stoppedErr = nil
	for _, startedC := range sb.startedCs {
		close(startedC)
	}
	sb.startedCs = make([]chan error, 0, 16)

	return nil
}

// Started returns a channel that will get closed once the boot process went
// successfully.
// This is thread-safe.
//
// Every failed start attempt will push an error (retries due to exponential
// backoff are not treated as failed attempts); hence you need to make sure
// to keep reading continuously on the returned channel or you might block
// the Maestro otherwise.
func (sb *ServiceBase) Started() <-chan error {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	// preallocate 32 to avoid deadlocks if the end-user is sloppy with their
	// goroutines... but not *too* sloppy though.
	errC := make(chan error, 32)
	if sb.started {
		close(errC)
	} else {
		sb.startedCs = append(sb.startedCs, errC)
	}
	return errC
}

func (sb *ServiceBase) stop(err error) error {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	/* one or more of our parents have failed to stop */
	if err != nil {
		sb.stopped = false
		sb.stoppedErr = err
		for _, stoppedC := range sb.stoppedCs {
			stoppedC <- err
		}
		return err
	}

	/* we're already stopped, this is a noop */
	if sb.stopped {
		return nil
	}

	/* successful shutdown */
	sb.stopped = true
	sb.stoppedErr = nil
	sb.started = !sb.stopped
	sb.startedErr = nil
	for _, stoppedC := range sb.stoppedCs {
		close(stoppedC)
	}
	sb.stoppedCs = make([]chan error, 0, 16)

	return nil
}

// Stopped returns a channel that will get closed once the shutdown process went
// successfully.
// This is thread-safe.
//
// Every failed shutdown attempt will push an error (retries due to exponential
// backoff are not treated as failed attempts); hence you need to make sure
// to keep reading continuously on the returned channel or you might block
// the Maestro otherwise.
func (sb *ServiceBase) Stopped() <-chan error {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	// preallocate 32 to avoid deadlocks if the end-user is sloppy with their
	// goroutines... but not *too* sloppy though.
	errC := make(chan error, 32)
	if sb.stopped {
		close(errC)
	} else {
		sb.stoppedCs = append(sb.stoppedCs, errC)
	}
	return errC
}
