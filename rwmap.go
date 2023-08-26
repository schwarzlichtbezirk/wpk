package wpk

import (
	"sync"
)

type RWMap[K comparable, T any] struct {
	m   map[K]T
	mux sync.RWMutex
}

func (rwm *RWMap[K, T]) Init() {
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	rwm.m = map[K]T{}
}

func (rwm *RWMap[K, T]) Len() int {
	rwm.mux.RLock()
	defer rwm.mux.RUnlock()
	return len(rwm.m)
}

func (rwm *RWMap[K, T]) Has(key K) (ok bool) {
	rwm.mux.RLock()
	defer rwm.mux.RUnlock()
	_, ok = rwm.m[key]
	return
}

func (rwm *RWMap[K, T]) Get(key K) (ret T, ok bool) {
	rwm.mux.RLock()
	defer rwm.mux.RUnlock()
	ret, ok = rwm.m[key]
	return
}

func (rwm *RWMap[K, T]) Set(key K, val T) {
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	rwm.m[key] = val
}

func (rwm *RWMap[K, T]) Delete(key K) {
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	delete(rwm.m, key)
}

func (rwm *RWMap[K, T]) GetAndDelete(key K) (ret T, ok bool) {
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	if ret, ok = rwm.m[key]; ok {
		delete(rwm.m, key)
	}
	return
}

func (rwm *RWMap[K, T]) Range(f func(K, T) bool) {
	type cell struct {
		key K
		val T
	}
	var buf []cell
	func() {
		rwm.mux.RLock()
		defer rwm.mux.RUnlock()
		buf = make([]cell, len(rwm.m))
		var i int
		for k, v := range rwm.m {
			buf[i].key, buf[i].val = k, v
			i++
		}
	}()
	for _, cell := range buf {
		if !f(cell.key, cell.val) {
			return
		}
	}
}
