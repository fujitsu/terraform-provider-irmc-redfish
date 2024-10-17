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
