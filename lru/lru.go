/*
	wanglei@ucloud.cn
*/

// Package lru implements LRU cache with concurrent safe / not safe version.
// adapted from groupCache
package lru

import (
	"container/list"
	"sync"
)

// Cache is an LRU cache. Not safe for concurrent access.
type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specificies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key Key, value interface{})

	ll    *list.List
	cache map[interface{}]*list.Element //链表反映时间次序 oldest 在最后
	// cache[key].Value 为 entry  cache[key].Value.value为真正value
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{}

type entry struct {
	key   Key
	value interface{}
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func NewCache(maxEntries int) *Cache {
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key Key, value interface{}) {
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*entry).value = value
		return
	}
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		c.RemoveOldest()
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key Key) (value interface{}, ok bool) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key Key) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
	if c.cache == nil {
		return
	}
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

// Remove specific element
func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

///////////////////////////////////////////////////////////////////////////////

// MuCache is an LRU cache. Safe for concurrent access.
// !! Sometimes It Is Better to Just Use Cache and make concurrent safe on your own
type MuCache struct {
	cache Cache
	sync.RWMutex
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func NewMuCache(maxEntries int) *MuCache {
	return &MuCache{
		cache: Cache{
			MaxEntries: maxEntries,
			ll:         list.New(),
			cache:      make(map[interface{}]*list.Element),
		},
	}
}

// Add adds a value to the cache.
func (m *MuCache) Add(key Key, value interface{}) {
	m.Lock()
	defer m.Unlock()
	m.cache.Add(key, value)
}

// Get looks up a key's value from the cache.
func (m *MuCache) Get(key Key) (value interface{}, ok bool) {
	m.RLock()
	defer m.RUnlock()
	return m.cache.Get(key)
}

// Remove removes the provided key from the cache.
func (m *MuCache) Remove(key Key) {
	m.Lock()
	defer m.Unlock()
	m.cache.Remove(key)
}

// RemoveOldest removes the oldest item from the cache.
func (m *MuCache) RemoveOldest() {
	m.Lock()
	defer m.Unlock()
	m.cache.RemoveOldest()
}

// Remove specific element
func (m *MuCache) removeElement(e *list.Element) {
	m.Lock()
	defer m.Unlock()
	m.cache.removeElement(e)
}

// Len returns the number of items in the cache.
func (m *MuCache) Len() int {
	m.RLock()
	defer m.RUnlock()
	return m.cache.Len()
}

// SetOnEvicted function
func (m *MuCache) SetOnEvicted(f func(key Key, value interface{})) {
	m.Lock()
	defer m.Unlock()
	m.cache.OnEvicted = f
}
