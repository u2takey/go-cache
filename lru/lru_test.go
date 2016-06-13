/*
	wanglei@ucloud.cn
*/

package lru

import (
	"math/rand"
	"sync"
	"testing"
)

type simpleStruct struct {
	int
	string
}

type complexStruct struct {
	int
	simpleStruct
}

var getTests = []struct {
	name       string
	keyToAdd   interface{}
	keyToGet   interface{}
	expectedOk bool
}{
	{"string_hit", "myKey", "myKey", true},
	{"string_miss", "myKey", "nonsense", false},
	{"simple_struct_hit", simpleStruct{1, "two"}, simpleStruct{1, "two"}, true},
	{"simeple_struct_miss", simpleStruct{1, "two"}, simpleStruct{0, "noway"}, false},
	{"complex_struct_hit", complexStruct{1, simpleStruct{2, "three"}},
		complexStruct{1, simpleStruct{2, "three"}}, true},
}

func TestGet(t *testing.T) {
	for _, tt := range getTests {
		lru := NewCache(0)
		lru.Add(tt.keyToAdd, 1234)
		val, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && val != 1234 {
			t.Fatalf("%s expected get to return 1234 but got %v", tt.name, val)
		}
	}
}

func TestRemove(t *testing.T) {
	lru := NewCache(0)
	lru.Add("myKey", 1234)
	if val, ok := lru.Get("myKey"); !ok {
		t.Fatal("TestRemove returned no match")
	} else if val != 1234 {
		t.Fatalf("TestRemove failed.  Expected %d, got %v", 1234, val)
	}

	lru.Remove("myKey")
	if _, ok := lru.Get("myKey"); ok {
		t.Fatal("TestRemove returned a removed entry")
	}
}

func TestOnEvicted(t *testing.T) {
	lru := NewMuCache(0)
	lru.Add("myKey", 1234)
	testMap := make(map[string]interface{}, 10)
	lru.SetOnEvicted(func(key Key, value interface{}) {
		testMap[key.(string)] = value.(int)
	})

	lru.Remove("myKey")
	if _, ok := lru.Get("myKey"); ok {
		t.Fatal("TestOnEvicted returned a removed entry")
	}

	value, ok := testMap["myKey"]
	if !ok {
		t.Fatal("TestOnEvicted onEvicted not call")
	}
	if value != 1234 {
		t.Fatal("TestOnEvicted onEvicted not call")
	}
}

func TestMaxEntry(t *testing.T) {
	lru := NewMuCache(10)
	for i := 0; i < 10; i++ {
		lru.Add(i, i)
	}
	// Get 1 will move 1 to front
	if _, ok := lru.Get(0); !ok {
		t.Fatal("TestMaxEntry Fail entries miss")
	}
	lru.Add(11, 11)

	// Add new one will remove 1
	if _, ok := lru.Get(0); !ok {
		t.Fatal("TestMaxEntry Fail entries miss")
	}
	if _, ok := lru.Get(1); ok {
		t.Fatal("TestMaxEntry Fail envict not ok")
	}
}

func BenchmarkCache(b *testing.B) {
	lru := NewCache(100000)
	rand.Seed(42)
	for n := 0; n < b.N; n++ {
		for j := 0; j < 100000; j++ {
			lru.Add(rand.Intn(100000), "just for test")
			lru.Get(rand.Intn(100000))
		}
	}

}

func BenchmarkMuCache(b *testing.B) {
	lru := NewMuCache(100000)
	rand.Seed(42)
	for n := 0; n < b.N; n++ {
		for j := 0; j < 100000; j++ {
			lru.Add(rand.Intn(100000), "just for test")
			lru.Get(rand.Intn(100000))
		}
	}
}

func BenchmarkMuCacheConcurrent(b *testing.B) {
	lru := NewMuCache(10000)
	rand.Seed(42)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < b.N; n++ {
				for j := 0; j < 100000; j++ {
					lru.Add(rand.Intn(100000), "just for test")
					lru.Get(rand.Intn(100000))
				}
			}
		}()
	}
	wg.Wait()
}
