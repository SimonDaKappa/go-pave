package pave

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test BindingCache functionality
func TestBindingCache(t *testing.T) {
	t.Run("NewBindingCache", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		assert.NotNil(t, cache)
	})

	t.Run("GetOrCreate", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source := "test"
		sourcePtr := &source // Use the same pointer throughout the test

		// First call should create
		entry1 := cache.GetOrCreate(sourcePtr, func() int { return 42 })
		assert.NotNil(t, entry1)
		assert.Equal(t, 42, entry1.GetData())

		// Second call should return same entry (same pointer, same key)
		entry2 := cache.GetOrCreate(sourcePtr, func() int {
			t.Error("Factory function should not be called second time")
			return 99
		})

		// Check that we got the same entry back
		if entry1 != entry2 {
			t.Errorf("Expected same entry, got different ones. Entry1: %p (data: %d), Entry2: %p (data: %d)",
				entry1, entry1.GetData(), entry2, entry2.GetData())
		}

		// Check that the data is from the first call
		assert.Equal(t, 42, entry2.GetData(), "Second call should return entry with data from first call")
	})

	t.Run("Get", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source := "test"
		sourcePtr := &source

		// Should not exist initially
		entry, exists := cache.Get(sourcePtr)
		assert.Nil(t, entry)
		assert.False(t, exists)

		// Create entry
		created := cache.GetOrCreate(sourcePtr, func() int { return 42 })

		// Should exist now
		entry, exists = cache.Get(sourcePtr)
		assert.NotNil(t, entry)
		assert.True(t, exists)
		assert.Equal(t, created, entry)
	})

	t.Run("Delete", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source := "test"
		sourcePtr := &source

		// Create entry
		cache.GetOrCreate(sourcePtr, func() int { return 42 })

		// Verify it exists
		entry, exists := cache.Get(sourcePtr)
		assert.True(t, exists)
		assert.NotNil(t, entry)

		// Delete it
		cache.Delete(sourcePtr)

		// Should not exist now
		entry, exists = cache.Get(sourcePtr)
		assert.False(t, exists)
		assert.Nil(t, entry)
	})

	t.Run("Clear", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source1 := "test1"
		source2 := "test2"
		sourcePtr1 := &source1
		sourcePtr2 := &source2

		// Create multiple entries
		cache.GetOrCreate(sourcePtr1, func() int { return 42 })
		cache.GetOrCreate(sourcePtr2, func() int { return 99 })

		// Verify they exist
		entry1, exists1 := cache.Get(sourcePtr1)
		entry2, exists2 := cache.Get(sourcePtr2)
		assert.True(t, exists1)
		assert.True(t, exists2)
		assert.NotNil(t, entry1)
		assert.NotNil(t, entry2)

		// Clear all
		cache.Clear()

		// Should not exist now
		entry1, exists1 = cache.Get(sourcePtr1)
		entry2, exists2 = cache.Get(sourcePtr2)
		assert.False(t, exists1)
		assert.False(t, exists2)
		assert.Nil(t, entry1)
		assert.Nil(t, entry2)
	})
}

// Test CacheEntry functionality
func TestCacheEntry(t *testing.T) {
	t.Run("ReadData", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source := "test"
		sourcePtr := &source

		entry := cache.GetOrCreate(sourcePtr, func() int { return 42 })

		var readValue int
		entry.ReadData(func(data int) {
			readValue = data
		})

		assert.Equal(t, 42, readValue)
	})

	t.Run("WriteData", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source := "test"
		sourcePtr := &source

		entry := cache.GetOrCreate(sourcePtr, func() int { return 42 })

		// Write new value
		entry.WriteData(func(data *int) {
			*data = 99
		})

		// Verify it was written
		assert.Equal(t, 99, entry.GetData())
	})

	t.Run("GetData", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source := "test"
		sourcePtr := &source

		entry := cache.GetOrCreate(sourcePtr, func() int { return 42 })

		data := entry.GetData()
		assert.Equal(t, 42, data)
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		cache := NewBindingCache[string, int]()
		source := "test"
		sourcePtr := &source

		entry := cache.GetOrCreate(sourcePtr, func() int { return 0 })

		// Test concurrent read/write access
		done := make(chan bool, 2)

		// Writer goroutine
		go func() {
			for i := 0; i < 100; i++ {
				entry.WriteData(func(data *int) {
					*data = i
				})
			}
			done <- true
		}()

		// Reader goroutine
		go func() {
			for i := 0; i < 100; i++ {
				entry.ReadData(func(data int) {
					// Just read the data, don't need to verify specific value
					// due to concurrent access
					_ = data
				})
			}
			done <- true
		}()

		// Wait for both goroutines
		<-done
		<-done

		// Test should not panic or race
		finalValue := entry.GetData()
		assert.True(t, finalValue >= 0 && finalValue < 100)
	})
}
