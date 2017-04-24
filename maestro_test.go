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
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

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

func TestMaestro_AddService_Service(t *testing.T) {
	/* already exists (panic) */
	/* must inherit (panic) */
	/* success */
}

func TestMaestro_StartAll_StopAll(t *testing.T) {
	/* missing deps (error) */
	/* circular deps (error) */
	/* failure (dep failed) */
	/* success */
}
