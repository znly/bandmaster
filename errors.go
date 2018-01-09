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

// ErrorKind enumerates the possible kind of errors returned by BandMaster.
type ErrorKind int

const (
	/* common */
	ErrUnknown ErrorKind = iota // unknown error

	/* fatal */
	ErrServiceAlreadyExists       ErrorKind = iota // service already exists
	ErrServiceWithoutBase         ErrorKind = iota // service must inherit form ServiceBase
	ErrServiceDependsOnItself     ErrorKind = iota // service depends on its own self
	ErrServiceDuplicateDependency ErrorKind = iota // service has duplicate dependencies
	ErrServicePropsInvalid        ErrorKind = iota // couldn't parse service properties from environment

	/* dependencies */
	ErrDependencyMissing     ErrorKind = iota // no such dependency
	ErrDependencyCircular    ErrorKind = iota // circular dependencies detected
	ErrDependencyUnavailable ErrorKind = iota // dependency failed to start
	ErrParentUnavailable     ErrorKind = iota // parent failed to shutdown

	/* runtime */
	ErrServiceStartFailure ErrorKind = iota // service failed to start
	ErrServiceStopFailure  ErrorKind = iota // service failed to stop
)

// Error is an error returned by BandMaster.
//
// Only the `Kind` field is guaranteed to be defined at all times; the rest
// may or may not be set depending on the kind of the returned error.
type Error struct {
	Kind ErrorKind

	Service     Service
	ServiceName string
	ServiceErr  error

	Dependency   string
	Parent       string
	CircularDeps []string
}

func (e *Error) Error() string {
	switch e.Kind {
	/* Common */
	case ErrUnknown:
		return "error: unknown"

	/* Fatal */
	case ErrServiceAlreadyExists:
		return fmt.Sprintf("`%s`: service already exists", e.ServiceName)
	case ErrServiceWithoutBase:
		return fmt.Sprintf("`%s`: service *must* inherit from `ServiceBase`", e.ServiceName)
	case ErrServiceDependsOnItself:
		return fmt.Sprintf("`%s`: service depends on its own self",
			e.ServiceName,
		)
	case ErrServiceDuplicateDependency:
		return fmt.Sprintf("`%s`: service already depends on `%s`",
			e.ServiceName, e.Dependency,
		)

	/* Dependencies */
	case ErrDependencyMissing:
		return fmt.Sprintf("`%s`: missing dependency `%s`",
			e.Service.Name(), e.Dependency,
		)
	case ErrDependencyCircular:
		return fmt.Sprintf("`%s`: circular dependencies detected: {%s}",
			e.Service.Name(), strings.Join(e.CircularDeps, " -> "),
		)
	case ErrDependencyUnavailable:
		return fmt.Sprintf("`%s`: dependency `%s` failed to start",
			e.Service.Name(), e.Dependency,
		)

	/* Runtime */
	case ErrServiceStartFailure:
		return fmt.Sprintf("`%s`: service failed to start: %s",
			e.Service.Name(), e.ServiceErr,
		)
	case ErrServiceStopFailure:
		return fmt.Sprintf("`%s`: service failed to stop: %s",
			e.Service.Name(), e.ServiceErr,
		)

	default:
		return "error: undefined"
	}
}
