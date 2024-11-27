package util

import (
	"sync"
)

type kvpair[K comparable, T any] struct {
	key K
	val T
}

// SeqMap is thread-safe map with key-value pairs sequence
// in which they were placed into map.
type SeqMap[K comparable, T any] struct {
	seq []kvpair[K, T]
	idx map[K]int
	mux sync.RWMutex
}

func (m *SeqMap[K, T]) Init(c int) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.idx = make(map[K]int, c)
	m.seq = make([]kvpair[K, T], 0, c)
}

func (m *SeqMap[K, T]) Len() int {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return len(m.seq)
}

func (m *SeqMap[K, T]) Has(key K) (ok bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	_, ok = m.idx[key]
	return
}

func (m *SeqMap[K, T]) Peek(key K) (ret T, ok bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	var n int
	if n, ok = m.idx[key]; ok {
		ret = m.seq[n].val
	}
	return
}

func (m *SeqMap[K, T]) Poke(key K, val T) {
	m.mux.Lock()
	defer m.mux.Unlock()

	var n, ok = m.idx[key]
	if ok {
		m.seq[n].val = val
	} else {
		m.idx[key] = len(m.seq)
		m.seq = append(m.seq, kvpair[K, T]{
			key: key,
			val: val,
		})
	}
}

func (m *SeqMap[K, T]) Pop() (key K, val T, ok bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if n := len(m.seq) - 1; n >= 0 {
		key, val, ok = m.seq[n].key, m.seq[n].val, true
		delete(m.idx, key)
		m.seq = m.seq[:n]
	}
	return
}

func (m *SeqMap[K, T]) Push(key K, val T) (ok bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, ok = m.idx[key]; !ok {
		m.idx[key] = len(m.seq)
		m.seq = append(m.seq, kvpair[K, T]{
			key: key,
			val: val,
		})
	}
	return
}

func (m *SeqMap[K, T]) Delete(key K) (ret T, ok bool) {
	var n int

	m.mux.Lock()
	defer m.mux.Unlock()

	if n, ok = m.idx[key]; ok {
		ret = m.seq[n].val
		delete(m.idx, key)
		copy(m.seq[n:], m.seq[n+1:])
		m.seq = m.seq[:len(m.seq)-1]
		for i := n; i < len(m.seq); i++ {
			m.idx[m.seq[i].key] = i
		}
	}
	return
}

func (m *SeqMap[K, T]) Range(f func(K, T) bool) {
	m.mux.Lock()
	var s = make([]kvpair[K, T], len(m.seq))
	copy(s, m.seq)
	m.mux.Unlock()

	for _, pair := range s {
		if !f(pair.key, pair.val) {
			return
		}
	}
}
