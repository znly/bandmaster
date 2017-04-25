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
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------

func TestMaestro_GlobalInstance(t *testing.T) {
	t.Run("replacement", func(t *testing.T) {
		gm := GlobalMaestro()
		assert.NotNil(t, GlobalMaestro())
		assert.Equal(t, fmt.Sprintf("%p", gm), fmt.Sprintf("%p", GlobalMaestro()))
		m := NewMaestro()
		assert.Equal(t, fmt.Sprintf("%p", gm), fmt.Sprintf("%p", ReplaceGlobalMaestro(m)))
		assert.Equal(t, fmt.Sprintf("%p", m), fmt.Sprintf("%p", GlobalMaestro()))
	})

	t.Run("parallel", func(t *testing.T) {
		nbRoutines := 128 * runtime.GOMAXPROCS(0)
		wg := &sync.WaitGroup{}
		wg.Add(nbRoutines)
		for i := 0; i < nbRoutines; i++ {
			go func(ii int) {
				defer wg.Done()
				end := time.After(time.Second)
				gm := GlobalMaestro()
				for {
					select {
					case <-end:
						return
					default:
						if rand.Intn(2) == 0 {
							gm = GlobalMaestro()
						} else {
							gm = ReplaceGlobalMaestro(gm)
						}
					}
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	})
}

// -----------------------------------------------------------------------------

func TestMaestro_AddService_Service(t *testing.T) {
	m := NewMaestro()

	t.Run("already-exists", func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				assert.NotNil(t, err)
				assert.IsType(t, &Error{}, err)
				assert.Equal(t,
					&Error{kind: ErrServiceAlreadyExists, serviceName: "A"}, err,
				)
			}
		}()
		m.AddService("A", true, &TestService{ServiceBase: NewServiceBase()})
		m.AddService("A", false, &TestService{ServiceBase: NewServiceBase()})
	})

	t.Run("must-inherit", func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				assert.NotNil(t, err)
				assert.IsType(t, &Error{}, err)
				assert.Equal(t,
					&Error{kind: ErrServiceWithoutBase, serviceName: "B"}, err,
				)
			}
		}()
		m.AddService("B", true, &TestService{})
	})

	t.Run("success", func(t *testing.T) {
		s := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("B", true, s)
		assert.Equal(t, s, m.Service("B"))
	})

	t.Run("parallel", func(t *testing.T) {
		nbRoutines := 128 * runtime.GOMAXPROCS(0)
		m := NewMaestro()
		wg := &sync.WaitGroup{}
		wg.Add(nbRoutines)
		for i := 0; i < nbRoutines; i++ {
			go func(ii int) {
				defer wg.Done()
				end := time.After(time.Second)
				for {
					select {
					case <-end:
						return
					default:
						s := &TestService{ServiceBase: NewServiceBase()}
						name := strconv.Itoa(ii) + strconv.Itoa(int(rand.Int63()))
						m.AddService(name, true, s)
						assert.Equal(t, s, m.Service(name))
					}
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	})
}

func TestMaestro_StartAll_StopAll(t *testing.T) {
	t.Run("missing-deps", func(t *testing.T) {
		m := NewMaestro()
		m.AddService("A", true, &TestService{ServiceBase: NewServiceBase()})
		m.AddService("B", true, &TestService{ServiceBase: NewServiceBase()}, "A")
		s := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("C", true, s, "A", "B", "D")
		err := errors.Cause(<-m.StartAll(context.Background()))
		errExpected := &Error{kind: ErrDependencyMissing, service: s, dependency: "D"}
		assert.Equal(t, errExpected, err)
	})

	t.Run("circular-deps", func(t *testing.T) {
		m := NewMaestro()
		a := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("A", true, a, "D")
		b := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("B", true, b, "A")
		c := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("C", true, c, "D")
		d := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("D", true, d, "B")
		err := errors.Cause(<-m.StartAll(context.Background()))
		assert.IsType(t, &Error{}, err)
		e := err.(*Error)
		errExpected := &Error{kind: ErrDependencyCircular}
		switch e.service.Name() {
		case "A":
			errExpected.service = a
			errExpected.circularDeps = []string{"A", "D", "B", "A"}
		case "B":
			errExpected.service = b
			errExpected.circularDeps = []string{"B", "A", "D", "B"}
		case "C":
			errExpected.service = c
			errExpected.circularDeps = []string{"C", "D", "B", "A", "D"}
		case "D":
			errExpected.service = d
			errExpected.circularDeps = []string{"D", "B", "A", "D"}
		default:
			assert.Fail(t, "should never get here")
		}
		assert.Equal(t, errExpected, err)
		assert.False(t, a.started)
		assert.False(t, a.stopped)
		assert.False(t, b.started)
		assert.False(t, b.stopped)
		assert.False(t, c.started)
		assert.False(t, c.stopped)
		assert.False(t, d.started)
		assert.False(t, d.stopped)
	})

	t.Run("success", func(t *testing.T) {
		m := NewMaestro()

		a := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("A", true, a)
		b := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("B", true, b, "A")
		c := &TestService{ServiceBase: NewServiceBase()}
		m.AddService("C", true, c, "A", "B")

		assert.Nil(t, <-m.StartAll(context.Background()))
		assert.True(t, a.started)
		assert.False(t, a.stopped)
		assert.Nil(t, <-a.Started())
		assert.True(t, b.started)
		assert.False(t, b.stopped)
		assert.Nil(t, <-b.Started())
		assert.True(t, c.started)
		assert.False(t, c.stopped)
		assert.Nil(t, <-c.Started())

		assert.Nil(t, <-m.StopAll(context.Background()))
		assert.True(t, a.started)
		assert.True(t, a.stopped)
		assert.Nil(t, <-a.Started())
		assert.Nil(t, <-a.Stopped())
		assert.True(t, b.started)
		assert.True(t, b.stopped)
		assert.Nil(t, <-b.Started())
		assert.Nil(t, <-b.Stopped())
		assert.True(t, c.started)
		assert.True(t, c.stopped)
		assert.Nil(t, <-c.Started())
		assert.Nil(t, <-c.Stopped())
	})
}
