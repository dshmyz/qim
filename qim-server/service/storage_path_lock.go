package service

import "sync"

type storagePathLockEntry struct {
	mu   sync.Mutex
	uses int
}

var storagePathLocks = struct {
	sync.Mutex
	entries map[string]*storagePathLockEntry
}{entries: make(map[string]*storagePathLockEntry)}

// withStoragePathLock serializes all lifetime decisions for one physical
// storage path. Entries are removed once the last caller releases the lock so
// the keyed mutex map cannot grow with every uploaded file.
func withStoragePathLock(path string, fn func() error) error {
	if path == "" {
		return fn()
	}

	storagePathLocks.Lock()
	entry := storagePathLocks.entries[path]
	if entry == nil {
		entry = &storagePathLockEntry{}
		storagePathLocks.entries[path] = entry
	}
	entry.uses++
	storagePathLocks.Unlock()

	entry.mu.Lock()
	defer func() {
		entry.mu.Unlock()

		storagePathLocks.Lock()
		entry.uses--
		if entry.uses == 0 {
			delete(storagePathLocks.entries, path)
		}
		storagePathLocks.Unlock()
	}()

	return fn()
}
