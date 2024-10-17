package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Synchronization must be done per system in pool, so every system can be controlled separately
// for that reason we need ~container for keeping mutexes.
type SyncPool struct {
	lock sync.Mutex
	pool map[string]*sync.Mutex
}

func InitSyncPoolInstance() *SyncPool {
	return &SyncPool{
		pool: make(map[string]*sync.Mutex),
	}
}

func (sp *SyncPool) getEndpointMutex(endpoint string) *sync.Mutex {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	mutex, ok := sp.pool[endpoint]
	if !ok {
		mutex = &sync.Mutex{}
		sp.pool[endpoint] = mutex
	}
	return mutex
}

func (sp *SyncPool) Lock(ctx context.Context, endpoint string, resource string) {
	var msg string
	msg = fmt.Sprintf("Before locking mutex for endpoint '%s', resource '%s'", endpoint, resource)
	tflog.Info(ctx, msg)

	sp.getEndpointMutex(endpoint).Lock()

	msg = fmt.Sprintf("Successfully locked mutex for endpoint '%s', resource '%s'", endpoint, resource)
	tflog.Info(ctx, msg)
}

func (sp *SyncPool) Unlock(ctx context.Context, endpoint string, resource string) {
	var msg string
	msg = fmt.Sprintf("Before unlocking mutex for endpoint '%s', resource '%s'", endpoint, resource)
	tflog.Info(ctx, msg)

	sp.getEndpointMutex(endpoint).Unlock()

	msg = fmt.Sprintf("Successfully unlocked mutex for endpoint '%s', resource '%s'", endpoint, resource)
	tflog.Info(ctx, msg)
}
