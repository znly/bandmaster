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
					&Error{Kind: ErrServiceAlreadyExists, ServiceName: "A"}, err)
			}
		}()
		m.AddService("A", true, NewTestService())
		m.AddService("A", false, NewTestService())
	})

	t.Run("must-inherit", func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				assert.NotNil(t, err)
				assert.IsType(t, &Error{}, err)
				assert.Equal(t,
					&Error{Kind: ErrServiceWithoutBase, ServiceName: "B"}, err)
			}
		}()
		m.AddService("B", true, &TestService{})
	})

	t.Run("depends-on-itself", func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				assert.NotNil(t, err)
				assert.IsType(t, &Error{}, err)
				assert.Equal(t, &Error{
					Kind: ErrServiceDependsOnItself, ServiceName: "X"}, err)
			}
		}()
		m.AddService("X", true, NewTestService(), "X")
	})

	t.Run("duplicate-dependency", func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				assert.NotNil(t, err)
				assert.IsType(t, &Error{}, err)
				assert.Equal(t, &Error{
					Kind:        ErrServiceDuplicateDependency,
					ServiceName: "X",
					Dependency:  "B",
				}, err)
			}
		}()
		m.AddService("X", true, NewTestService(), "B", "B")
	})

	t.Run("success", func(t *testing.T) {
		s := NewTestService()
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
						s := NewTestService()
						name := strconv.Itoa(ii) + strconv.Itoa(int(rand.Int63()))
						m.AddService(name, true, s)
						assert.Equal(t, s, m.Service(name))
					}
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestMaestro_StartAll_StopAll(t *testing.T) {
	t.Run("missing-deps", func(t *testing.T) {
		m := NewMaestro()
		m.AddService("A", true, NewTestService())
		m.AddService("B", true, NewTestService(), "A")
		s := NewTestService()
		m.AddService("C", true, s, "A", "B", "D")
		errExpected := &Error{
			Kind: ErrDependencyMissing, Service: s, Dependency: "D"}

		/* heavily cautious parallel status checks */
		wg1 := &sync.WaitGroup{}
		for i := 0; i < 50; i++ {
			wg1.Add(1)
			go func() {
				defer wg1.Done()

				assert.Equal(t, errExpected,
					errors.Cause(<-m.StartAll(context.Background())))

				select {
				case <-s.Started():
					assert.FailNow(t, "shouldn't be here")
				default:
					assert.NoError(t, <-s.Stopped())
				}
			}()
		}
		wg1.Wait()

		/* heavily cautious parallel status checks */
		wg2 := &sync.WaitGroup{}
		for i := 0; i < 50; i++ {
			wg2.Add(1)
			go func() {
				defer wg2.Done()

				assert.NoError(t, <-m.StopAll(context.Background()))

				select {
				case <-s.Started():
					assert.FailNow(t, "shouldn't be here")
				default:
					assert.NoError(t, <-s.Stopped())
				}
			}()
		}
		wg2.Wait()
	})

	t.Run("circular-deps", func(t *testing.T) {
		m := NewMaestro()
		a := NewTestService()
		m.AddService("A", true, a, "D")
		b := NewTestService()
		m.AddService("B", true, b, "A")
		c := NewTestService()
		m.AddService("C", true, c, "D")
		d := NewTestService()
		m.AddService("D", true, d, "B")

		/* heavily cautious parallel status checks */
		ss := []Service{a, b, c, d}
		wg1 := &sync.WaitGroup{}
		for i := 0; i < 50; i++ {
			for _, s := range ss {
				wg1.Add(1)
				go func(s Service) {
					defer wg1.Done()

					errExpected := &Error{Kind: ErrDependencyCircular}
					err := errors.Cause(<-m.StartAll(context.Background()))
					assert.IsType(t, &Error{}, err)
					e := err.(*Error)
					switch e.Service.Name() {
					case "A":
						errExpected.Service = a
						errExpected.CircularDeps = []string{"A", "D", "B", "A"}
					case "B":
						errExpected.Service = b
						errExpected.CircularDeps = []string{"B", "A", "D", "B"}
					case "C":
						errExpected.Service = c
						errExpected.CircularDeps = []string{"C", "D", "B", "A", "D"}
					case "D":
						errExpected.Service = d
						errExpected.CircularDeps = []string{"D", "B", "A", "D"}
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

					select {
					case <-s.Started():
						assert.FailNow(t, "shouldn't be here")
					default:
						assert.NoError(t, <-s.Stopped())
					}
				}(s)
			}
		}
		wg1.Wait()
		/* heavily cautious parallel status checks */
		wg2 := &sync.WaitGroup{}
		for i := 0; i < 50; i++ {
			for _, s := range ss {
				wg2.Add(1)
				go func(s Service) {
					defer wg2.Done()
					select {
					case <-s.Started():
						assert.FailNow(t, "shouldn't be here")
					default:
						assert.NoError(t, <-s.Stopped())
					}
					assert.NoError(t, <-m.StopAll(context.Background()))
					select {
					case <-s.Started():
						assert.FailNow(t, "shouldn't be here")
					default:
						assert.NoError(t, <-s.Stopped())
					}
				}(s)
			}
		}
		wg2.Wait()
	})

	t.Run("success", func(t *testing.T) {
		m := NewMaestro()

		a := NewTestService()
		m.AddService("A", true, a)
		b := NewTestService()
		m.AddService("B", true, b, "A")
		c := NewTestService()
		m.AddService("C", true, c, "A", "B")

		ctx := context.Background()

		wg1 := &sync.WaitGroup{}
		wg1.Add(1)
		go func() {
			defer wg1.Done()
			s := m.ServiceReady(ctx, "C")
			assert.Equal(t, c, s)
		}()

		/* heavily cautious parallel status checks */
		wg2 := &sync.WaitGroup{}
		ss := []*TestService{a, b, c}
		for i := 0; i < 50; i++ {
			for _, s := range ss {
				wg2.Add(1)
				go func(s *TestService) {
					defer wg2.Done()

					assert.Nil(t, <-m.StartAll(ctx))

					assert.True(t, s.started)
					assert.False(t, s.stopped)

					select {
					case <-s.Stopped():
						assert.FailNow(t, "shouldn't be here")
					default:
						assert.NoError(t, <-s.Started())
					}
				}(s)
			}
		}
		wg1.Wait()
		wg2.Wait()
		/* heavily cautious parallel status checks */
		wg3 := &sync.WaitGroup{}
		for i := 0; i < 50; i++ {
			for _, s := range ss {
				wg3.Add(1)
				go func(s *TestService) {
					defer wg3.Done()

					assert.Nil(t, <-m.StopAll(ctx))

					assert.True(t, s.started)
					assert.True(t, s.stopped)
					select {
					case <-s.Started():
						assert.FailNow(t, "shouldn't be here")
					default:
						assert.NoError(t, <-s.Stopped())
					}
				}(s)
			}
		}
		wg3.Wait()
	})

	t.Run("retry-backoff", func(t *testing.T) {
		m := NewMaestro()

		a := NewTestService()
		a.dontStart = true
		b := NewTestService()
		b.dontStart = true
		m.AddServiceWithBackoff("A", true, 10, time.Millisecond*500, a)
		m.AddServiceWithBackoff("B", true, 1, time.Millisecond*100, b)

		ctx := context.Background()

		wg1 := &sync.WaitGroup{}
		wg1.Add(2)
		go func() {
			defer wg1.Done()
			time.Sleep(time.Second * 2)
			a.lock.Lock()
			a.dontStart = false
			a.lock.Unlock()
			assert.Equal(t, a, m.ServiceReady(ctx, "A"))
		}()
		go func() {
			defer wg1.Done()
			time.Sleep(time.Second * 2)
			b.lock.Lock()
			b.dontStart = false
			b.lock.Unlock()
			ctx1, ctx1cancel := context.WithTimeout(ctx, time.Second*1)
			assert.Nil(t, m.ServiceReady(ctx1, "B"))
			ctx1cancel()
		}()
		errExpected := &Error{
			Kind:       ErrServiceStartFailure,
			Service:    b,
			ServiceErr: _testServiceErrNotAllowedToStart,
		}
		assert.Equal(t, errExpected, <-m.StartAll(ctx))
		wg1.Wait()

		wg2 := &sync.WaitGroup{}
		for i := 0; i < 50; i++ {
			wg2.Add(1)
			go func() {
				defer wg2.Done()

				assert.Nil(t, <-m.StopAll(ctx))

				select {
				case <-a.Started():
					assert.FailNow(t, "shouldn't be here")
				default:
					assert.NoError(t, <-a.Stopped())
				}
			}()
		}
		wg2.Wait()
	})
}
