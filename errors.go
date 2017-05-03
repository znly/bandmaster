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
	"fmt"
	"strings"
)

// -----------------------------------------------------------------------------

// TODO(cmc)
type ErrorKind int

const (
	/* common */
	ErrUnknown ErrorKind = iota // unknown error

	/* fatal */
	ErrServiceAlreadyExists       ErrorKind = iota // service already exists
	ErrServiceWithoutBase         ErrorKind = iota // service must inherit form ServiceBase
	ErrServiceDependsOnItself     ErrorKind = iota // service depends on its own self
	ErrServiceDuplicateDependency ErrorKind = iota // service has duplicate dependencies

	/* dependencies */
	ErrDependencyMissing     ErrorKind = iota // no such dependency
	ErrDependencyCircular    ErrorKind = iota // circular dependencies detected
	ErrDependencyUnavailable ErrorKind = iota // dependency failed to start

	/* runtime */
	ErrServiceStartFailure ErrorKind = iota // service failed to start
	ErrServiceStopFailure  ErrorKind = iota // service failed to stop
)

// TODO(cmc)
type Error struct {
	kind         ErrorKind
	service      Service
	serviceName  string
	serviceErr   error
	dependency   string
	circularDeps []string
}

func (e *Error) Service() Service { return e.service }

func (e *Error) Error() string {
	switch e.kind {
	/* Common */
	case ErrUnknown:
		return "error: unknown"

	/* Fatal */
	case ErrServiceAlreadyExists:
		return fmt.Sprintf("`%s`: service already exists", e.serviceName)
	case ErrServiceWithoutBase:
		return fmt.Sprintf("`%s`: service *must* inherit from `ServiceBase`", e.serviceName)
	case ErrServiceDependsOnItself:
		return fmt.Sprintf("`%s`: service depends on its own self",
			e.serviceName,
		)
	case ErrServiceDuplicateDependency:
		return fmt.Sprintf("`%s`: service already depends on `%s`",
			e.serviceName, e.dependency,
		)

	/* Dependencies */
	case ErrDependencyMissing:
		return fmt.Sprintf("`%s`: missing dependency `%s`",
			e.service.Name(), e.dependency,
		)
	case ErrDependencyCircular:
		return fmt.Sprintf("`%s`: circular dependencies detected: {%s}",
			e.service.Name(), strings.Join(e.circularDeps, " -> "),
		)
	case ErrDependencyUnavailable:
		return fmt.Sprintf("`%s`: dependency `%s` failed to start",
			e.service.Name(), e.dependency,
		)

	/* Runtime */
	case ErrServiceStartFailure:
		return fmt.Sprintf("`%s`: service failed to start: %s",
			e.service.Name(), e.serviceErr,
		)
	case ErrServiceStopFailure:
		return fmt.Sprintf("`%s`: service failed to stop: %s",
			e.service.Name(), e.serviceErr,
		)

	default:
		return "error: undefined"
	}
}
