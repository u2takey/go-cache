/*
	wanglei@ucloud.cn
*/

package ttlCache2

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"
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
	key   string
	value interface{}
	err   error
}{
	{"myKey", "myKey", nil},
	{"myKey", "nonsense", nil},
	{"simpleStruct", simpleStruct{1, "two"}, nil},
	{"complexStruct", complexStruct{1, simpleStruct{2, "three"}}, errors.New("test")},
}

// test get key
func TestGet(t *testing.T) {
	for _, tt := range getTests {
		f := func(key string) (value interface{}, err error) {
			fmt.Println("get key", key)
			return tt.value, nil
		}
		cache := NewCache(360, 0, "", f)
		val, err := cache.Get(tt.key)
		if err != nil && err != tt.err {
			t.Fatalf("cache hit = %s; want %s", err, tt.err)
		} else if err == nil && val != tt.value {
			t.Fatalf("cache hit = %v; want %v", val, tt.value)
		}
	}
}

// test get key 2 --- key already exsit
func TestGet2(t *testing.T) {
	key := "1"
	value := 1
	f := func(key string) (value interface{}, err error) {
		t.Fatalf("key already exsit")
		return nil, nil
	}
	cache := NewCache(360, 0, "", f)
	cache.Add(key, value)
	val, err := cache.Get(key)
	if err != nil || val != value {
		t.Fatalf("get key fail", err)
	}
}

// test get key 3 --- key outof date
func TestGet3(t *testing.T) {
	key := "1"
	value := 1
	value2 := 2
	var counter uint64 = 0
	f := func(key string) (value interface{}, err error) {
		fmt.Println("geting key", key)
		time.Sleep(300 * time.Millisecond)
		atomic.AddUint64(&counter, 1)
		if counter > 1 {
			t.Fatalf("update should call once")
		}
		return 2, nil
	}
	cache := NewCache(1, 0, "", f)
	cache.Add(key, value)
	// old value
	for i := 0; i < 10; i++ {
		go func() {
			val, err := cache.Get(key)
			if err != nil || val != value {
				t.Fatalf("get key fail want : %d", value)
			}
		}()
	}
	// out of date but still old value
	time.Sleep(2 * time.Second)
	for i := 0; i < 10; i++ {
		go func() {
			val, err := cache.Get(key)
			fmt.Println("2. geting value", val)
			if err != nil || val != value {
				t.Fatalf("get key fail want : %d", value)
			}
		}()
	}
	// now new value
	time.Sleep(1 * time.Second)
	for j := 0; j < 10; j++ {
		go func() {
			val, err := cache.Get(key)
			fmt.Println("3. geting value", val)
			if err != nil || val != value2 {
				t.Fatalf("get key fail want : %d, get %d", value2, val)
			}
		}()
	}
	time.Sleep(1 * time.Second)
}

// test get key 4 --- key is nil and get key block
func TestGet4(t *testing.T) {
	key := "1"
	value := 1
	var counter uint64 = 0
	f := func(key string) (value interface{}, err error) {
		time.Sleep(300 * time.Millisecond)
		atomic.AddUint64(&counter, 1)
		fmt.Println("geting key", key, counter)
		return 1, nil
	}
	cache := NewCache(1, 0, "", f)

	// value is nil will block to get new value
	for i := 0; i < 10; i++ {
		go func() {
			val, err := cache.Get(key)
			fmt.Println("2. geting value", val)
			if err != nil || val != value {
				t.Fatalf("get key fail want : %d", value)
			}
		}()
	}

	// now new value
	time.Sleep(1 * time.Second)
	for j := 0; j < 10; j++ {
		go func() {
			val, err := cache.Get(key)
			fmt.Println("3. geting value", val)
			if err != nil || val != value {
				t.Fatalf("get key fail want : %d, get %d", value, val)
			}
		}()
	}
	time.Sleep(1 * time.Second)
}

// test flush
func TestFlush(t *testing.T) {
	os.Remove("cache.data")

	f := func(key string) (value interface{}, err error) {
		t.Fatalf("key already exsit")
		return nil, nil
	}
	cache := NewCache(360, 1, "cache.data", f)
	cache.Add("1", "2")
	cache.Add("2", "23e23e23e")
	time.Sleep(2 * time.Second)

	cache2 := NewCache(360, 10, "cache.data", f)
	val, err := cache2.Get("1")
	if err != nil || val != "2" {
		t.Fatalf("get key fail want : %s", "2")
	}
}
