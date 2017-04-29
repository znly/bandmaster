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
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------

func TestService_Metadata(t *testing.T) {
	ts := NewTestService()

	t.Run("name", func(t *testing.T) {
		ts.setName("bob")
		assert.Equal(t, "bob", ts.Name())
	})

	t.Run("required", func(t *testing.T) {
		ts.setRequired(true)
		assert.True(t, ts.Required())
	})

	t.Run("parallel", func(t *testing.T) {
		ts.setName("notempty")
		nbRoutines := 128 * runtime.GOMAXPROCS(0)
		wg := &sync.WaitGroup{}
		wg.Add(nbRoutines * 2)
		for i := 0; i < nbRoutines; i++ {
			go func(ii int) {
				defer wg.Done()
				end := time.After(time.Second)
				for {
					select {
					case <-end:
						return
					default:
						if rand.Intn(2) == 0 {
							ts.setName(time.Now().Format(time.RFC3339))
						} else {
							assert.NotEmpty(t, ts.Name())
						}
					}
				}
			}(i)
			go func(ii int) {
				defer wg.Done()
				end := time.After(time.Second)
				for {
					select {
					case <-end:
						return
					default:
						if rand.Intn(2) == 0 {
							ts.setName(time.Now().Format(time.RFC3339))
						} else {
							_ = ts.Required()
						}
					}
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestService_Dependencies(t *testing.T) {
	/* see also TestMaestro_AddService_Service/depends-on-itself */
	/* see also TestMaestro_AddService_Service/duplicate-dependency */

	t.Run("success", func(t *testing.T) {
		ts := NewTestService()
		ts.addDependency("X", "Y", "Z")
		assert.Equal(t,
			map[string]struct{}{"X": {}, "Y": {}, "Z": {}}, ts.Dependencies(),
		)
	})

	t.Run("parallel", func(t *testing.T) {
		ts := NewTestService()
		ts.addDependency("xxx")
		nbRoutines := 128 * runtime.GOMAXPROCS(0)
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
						if rand.Intn(2) == 0 {
							ts.addDependency(
								strconv.Itoa(ii) + strconv.Itoa(int(rand.Int63())),
							)
						} else {
							assert.NotEmpty(t, ts.Dependencies())
						}
					}
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestService_String(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		ts := NewTestService()
		assert.Equal(t, "'' [optional]", ts.String())
		ts.setName("xxx")
		assert.Equal(t, "'xxx' [optional]", ts.String())
		ts.setName("yyy")
		assert.Equal(t, "'yyy' [optional]", ts.String())
	})

	t.Run("parallel", func(t *testing.T) {
		ts := NewTestService()
		nbRoutines := 128 * runtime.GOMAXPROCS(0)
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
						if rand.Intn(2) == 0 {
							ts.setName(time.Now().Format(time.RFC3339))
						} else {
							assert.NotEmpty(t, ts.String())
						}
					}
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestService_State(t *testing.T) {
	/* already tested by `TestMaestro_StartAll_StopAll/success` */
}
