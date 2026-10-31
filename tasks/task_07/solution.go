package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}
type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		return value, false
	}
	c.mu.Lock()
	defer c.mu.Unlock() 
	elem, found := c.items[key]
	if found {
		c.ll.MoveToFront(elem)
		e := elem.Value.(entry[K, V])
		return e.value, true
	}
	return value, false
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 { 
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, found := c.items[key]
	if found {
		elem.Value = entry[K, V]{
			key: key, 
			value: value,
		}
		return
	}
	link := c.ll.PushFront(entry[K, V]{
		key: key,
		value: value,
	})
	c.items[key] = link
	if len(c.items) > c.capacity {
		lastElem  := c.ll.Back() 
		lastEntry := lastElem.Value.(entry[K, V])
		oldKey := lastEntry.key
		c.ll.Remove(lastElem)
		delete(c.items, oldKey)
	}
	return
}