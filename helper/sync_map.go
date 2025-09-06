package helper

import "sync"

type SyncMap[Key comparable, Value any] struct {
	inner sync.Map
}

func (m *SyncMap[Key, Value]) Get(key Key) (value Value, exists bool) {
	rawValue, exists := m.inner.Load(key)
	if !exists {
		return value, exists
	}
	return rawValue.(Value), exists
}

func (m *SyncMap[Key, Value]) Put(key Key, value Value) (old Value, exists bool) {
	rawValue, exists := m.inner.Swap(key, value)
	if !exists {
		return old, exists
	}
	return rawValue.(Value), exists
}

func (m *SyncMap[Key, Value]) Remove(key Key) (value Value, exists bool) {
	rawValue, exists := m.inner.LoadAndDelete(key)
	if !exists {
		return value, exists
	}
	return rawValue.(Value), exists
}

func (m *SyncMap[Key, Value]) PutIfAbsent(key Key, value Value) (actual Value, exists bool) {
	actualValue, exists := m.inner.LoadOrStore(key, value)
	if !exists {
		return value, exists
	}
	return actualValue.(Value), exists
}

func (m *SyncMap[Key, Value]) RemoveIfEquals(key Key, value Value) (exists bool) {
	return m.inner.CompareAndDelete(key, value)
}

func (m *SyncMap[Key, Value]) Replace(key Key, oldValue Value, newValue Value) (exists bool) {
	return m.inner.CompareAndSwap(key, oldValue, newValue)
}

func (m *SyncMap[Key, Value]) ForEach(f func(key Key, value Value) bool) {
	m.inner.Range(func(key, value any) bool { return f(key.(Key), value.(Value)) })
}
