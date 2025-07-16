package pave

import (
	"errors"
	"sync"
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

// GetOrCreate returns the cache entry for the source, creating one if it doesn't exist.
// The factory function is called only once per source instance, even under concurrent access.
func (bc *BindingCache[S, C]) GetOrCreate(source *S, factory func() C) *CacheEntry[C] {
	// Try to load existing entry
	if v, ok := bc.cache.Load(source); ok {
		return v.(*CacheEntry[C])
	}

	// Create new entry without calling factory yet
	newEntry := &CacheEntry[C]{}

	// LoadOrStore returns the actual stored value
	actual, loaded := bc.cache.LoadOrStore(source, newEntry)
	entry := actual.(*CacheEntry[C])

	// If we stored our new entry, initialize it
	if !loaded {
		entry.mutex.Lock()
		entry.data = factory()
		entry.mutex.Unlock()
	}

	return entry
}

// Get retrieves the cache entry for the source if it exists
func (bc *BindingCache[S, C]) Get(source *S) (*CacheEntry[C], bool) {
	if v, ok := bc.cache.Load(source); ok {
		return v.(*CacheEntry[C]), true
	}
	return nil, false
}

// Delete removes the cache entry for the source
func (bc *BindingCache[S, C]) Delete(source *S) {
	bc.cache.Delete(source)
}

// Clear removes all cache entries
func (bc *BindingCache[S, C]) Clear() {
	// Create a new map instead of iterating
	bc.cache = sync.Map{}
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
