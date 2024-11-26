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
