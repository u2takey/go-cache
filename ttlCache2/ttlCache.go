/*
	wanglei@ucloud.cn
*/

// Package ttlCache2
package ttlCache2

import (
	"encoding/json"
	//	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"
)

type UpdateKeyFunc func(key string) (value interface{}, err error)

// 1. Not like lru cache, entries in ttlCache has specific lifespan and
// get operation will not extend its lifespan.
// 2. Not like ttl cache, to keep entries avaliable, it will triger a
// update-key operation instead of purging entries out-of-date.
// 3. concurrent access safe
type Cache struct {
	// LifeTime is lifespan for all entries
	LifeTime uint64

	// auto flush cache to file if flushFile not "" and flushInterval > 0
	flushFile     string
	flushInterval uint64

	// update-key operation will be trigger when value is nil or value is out-of-date
	updateKeyFunc UpdateKeyFunc

	cache map[string]*entry
	sync.RWMutex
}

const (
	ready int32 = iota
	updating
)

type entry struct {
	Key   string
	Born  int64 // Unix time
	State int32
	Value interface{}
}

// New creates a new Cache.
// auto flush cache to file if flushFile not "" and flushInterval > 0
// for flush cache value must jsonable
func NewCache(lifeTime, flushInterval uint64, flushFile string, updateKeyFunc UpdateKeyFunc) *Cache {
	c := &Cache{
		LifeTime:      lifeTime,
		flushInterval: flushInterval,
		flushFile:     flushFile,
		updateKeyFunc: updateKeyFunc,
		cache:         make(map[string]*entry),
	}

	if flushFile != "" {
		c.load()
	}
	if flushFile != "" && flushInterval > 0 {
		ticker := time.NewTicker(time.Duration(flushInterval * 1000 * 1000 * 1000))
		go func() {
			for _ = range ticker.C {
				//				fmt.Println(
				c.flush()
				//				)
			}
		}()
	}
	return c
}

// Add adds a value to the cache. Update born and state if exsit
func (c *Cache) Add(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	//fmt.Println("Add", value)
	if c.cache == nil {
		c.cache = make(map[string]*entry)
	}
	if ee, ok := c.cache[key]; ok {
		ee.Value = value
		ee.Born = time.Now().Unix()
		ee.State = ready
		return
	}
	c.cache[key] = &entry{
		Key:   key,
		Value: value,
		Born:  time.Now().Unix(),
		State: ready,
	}
}

// Get looks up a key's value from the cache.
// if entry is nil will block util get value
// if entry is out of date will return old value and trigger an upate-key operation if needed
func (c *Cache) Get(key string) (value interface{}, err error) {
	c.RLock()
	if c.cache == nil {
		c.cache = make(map[string]*entry)
	}
	now := time.Now().Unix()

	ele, hit := c.cache[key]
	c.RUnlock()

	// hit
	if hit {
		// out of date and not updating
		if uint64(now-ele.Born) > c.LifeTime && atomic.CompareAndSwapInt32(&ele.State, ready, updating) {
			go func() {
				value, err := c.updateKeyFunc(key)
				if err == nil {
					c.Add(key, value)
				} else {
					atomic.SwapInt32(&ele.State, ready)
				}
			}()
		}
		//fmt.Println("return old value, out of date ", uint64(now-ele.born) > c.LifeTime)
		return ele.Value, nil
	} else { // not hit
		value, err := c.updateKeyFunc(key)
		if err == nil {
			c.Add(key, value)
			//fmt.Println("return new value")
			return value, nil
		}
	}
	return nil, err
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key string) {
	c.Lock()
	defer c.Unlock()
	if c.cache == nil {
		return
	}
	if _, hit := c.cache[key]; hit {
		delete(c.cache, key)
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.RLock()
	defer c.RUnlock()
	if c.cache == nil {
		return 0
	}
	return len(c.cache)
}

// load from file to cache
func (c *Cache) load() error {
	//fmt.Println("do load to ", c.flushFile, c.flushInterval)
	buf, err := ioutil.ReadFile(c.flushFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, &c.cache)
}

// flush cache to file
func (c *Cache) flush() error {
	//fmt.Println("do flush to ", c.flushFile, c.flushInterval)
	buf, err := json.Marshal(c.cache)
	if err != nil {
		return err
	}
	fmt.Println(string(buf))
	return ioutil.WriteFile(c.flushFile, buf, 0777)
}
