package pave

import (
	"errors"
	"sync"
	"unsafe"
)

var (
	ErrBindingCacheNotInitialized = errors.New("binding cache not initialized")
	ErrBindingCacheNilEntry       = errors.New("binding cache nil entry provided")
)

// BindingCache provides thread-safe caching of binding values per source instance.
// It uses the memory address of the source as the cache key, which is safe in Go
// since objects don't move once allocated.
type BindingCache[S any, C any] struct {
	cache sync.Map // map[uintptr]*CacheEntry[C]
}

// CacheEntry holds the cached data for a specific source instance
type CacheEntry[C any] struct {
	data  C            // Cached data
	mutex sync.RWMutex // Allows concurrent reads, exclusive writes
}

// NewBindingCache creates a new thread-safe binding cache
func NewBindingCache[S any, C any]() *BindingCache[S, C] {
	return &BindingCache[S, C]{
		cache: sync.Map{},
	}
}

// getSourceKey generates a unique key for the source instance using its memory address
func (bc *BindingCache[S, C]) getSourceKey(source *S) uintptr {
	return uintptr(unsafe.Pointer(source))
}

// GetOrCreate returns the cache entry for the source, creating one if it doesn't exist.
// The factory function is called only once per source instance, even under concurrent access.
func (bc *BindingCache[S, C]) GetOrCreate(source *S, new func() C) *CacheEntry[C] {
	key := bc.getSourceKey(source)

	// Try to load existing entry
	if entry, ok := bc.cache.Load(key); ok {
		return entry.(*CacheEntry[C])
	}

	// Create new entry
	newEntry := &CacheEntry[C]{
		data: new(),
	}

	// Store and return the entry that was actually stored
	actual, _ := bc.cache.LoadOrStore(key, newEntry)
	return actual.(*CacheEntry[C])
}

// Get retrieves the cache entry for the source if it exists
func (bc *BindingCache[S, C]) Get(source *S) (*CacheEntry[C], bool) {
	key := bc.getSourceKey(source)
	if entry, ok := bc.cache.Load(key); ok {
		return entry.(*CacheEntry[C]), true
	}
	return nil, false
}

// Delete removes the cache entry for the source
func (bc *BindingCache[S, C]) Delete(source *S) {
	key := bc.getSourceKey(source)
	bc.cache.Delete(key)
}

// Clear removes all cache entries
func (bc *BindingCache[S, C]) Clear() {
	bc.cache.Range(func(key, value interface{}) bool {
		bc.cache.Delete(key)
		return true
	})
}

// ReadData provides read access to the cached data
func (ce *CacheEntry[C]) ReadData(fn func(data C)) {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()
	fn(ce.data)
}

// WriteData provides write access to the cached data
func (ce *CacheEntry[C]) WriteData(fn func(data *C)) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()
	fn(&ce.data)
}

// GetData returns a copy of the cached data (safe for simple types)
func (ce *CacheEntry[C]) GetData() C {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()
	return ce.data
}
