/*
Copyright (c) 2024 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"sync"
	"testing"
)

func TestSyncPool(t *testing.T) {
	pool := InitSyncPoolInstance()
	var sum int = 0
	var items int = 50

	t.Run("MutexTests", func(t *testing.T) {
		var test = func() {
			wg := &sync.WaitGroup{}
			wg.Add(items)

			for i := 0; i < items; i++ {
				go func() {
					defer wg.Done()
					pool.Lock(context.TODO(), "", "")
					defer pool.Unlock(context.TODO(), "", "")
					sum += 1
				}()
			}

			wg.Wait()
		}

		test()

		if sum-items != 0 {
			t.Errorf("Got %d, expected %d", sum, items)
		}
	})
}
