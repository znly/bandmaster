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

import "fmt"

// -----------------------------------------------------------------------------

// TODO(cmc)
type ErrorKind int

const (
	/* common */
	ErrUnknown ErrorKind = iota // unknown error

	/* dependencies */
	ErrDependencyMissing     ErrorKind = iota // no such dependency
	ErrDependencyCircular    ErrorKind = iota // circular dependencies detected
	ErrDependencyUnavailable ErrorKind = iota // dependency failed to start

	/* runtime */
	ErrServiceStartFailure ErrorKind = iota // service failed to start
)

// TODO(cmc)
type Error struct {
	kind       ErrorKind
	service    Service
	serviceErr error
	dependency string
}

func (e *Error) Error() string {
	switch e.kind {
	/* common */
	case ErrUnknown:
		return "error: unknown"

	/* dependencies */
	case ErrDependencyMissing:
		return fmt.Sprintf("`%s`: missing dependency `%s`",
			e.service.Name(), e.dependency,
		)
	case ErrDependencyCircular:
		return fmt.Sprintf("`%s`: circular dependency with `%s` detected",
			e.service.Name(), e.dependency,
		)
	case ErrDependencyUnavailable:
		return fmt.Sprintf("`%s`: dependency `%s` failed to start",
			e.service.Name(), e.dependency,
		)

	/* runtime */
	case ErrServiceStartFailure:
		return fmt.Sprintf("`%s`: service failed to start: %s",
			e.service.Name(), e.serviceErr,
		)

	default:
		return "error: undefined"
	}
}