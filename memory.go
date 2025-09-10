package alchemist

import (
	"sync"
	"sync/atomic"
)

const arenaShards = 256

type PointerArena[T any] struct {
	shards  [arenaShards]*shard[T]
	counter atomic.Uintptr
}

type shard[T any] struct {
	mu sync.Mutex
	m  sync.Map // map[uintptr]*AlchemistValue[T]
}

func NewPointerArena[T any]() *PointerArena[T] {
	arena := &PointerArena[T]{}
	for i := 0; i < arenaShards; i++ {
		arena.shards[i] = &shard[T]{}
	}
	return arena
}

func (a *PointerArena[T]) shard(uid uintptr) *shard[T] {
	return a.shards[uid%arenaShards]
}

// Alloc возвращает уникальный UID и записывает объект
func (a *PointerArena[T]) alloc(obj *AlchemistValue[T]) uintptr {
	uid := a.counter.Add(1)
	obj.setUIDValue(uid)
	s := a.shard(uid)
	s.mu.Lock()
	s.m.Store(uid, obj)
	s.mu.Unlock()
	return uid
}

// Get возвращает объект по UID, lock-free
func (a *PointerArena[T]) Get(uid uintptr) *AlchemistValue[T] {
	s := a.shard(uid)
	if v, ok := s.m.Load(uid); ok {
		return v.(*AlchemistValue[T])
	}
	return nil
}

// Free удаляет объект
func (a *PointerArena[T]) free(uid uintptr) {
	s := a.shard(uid)
	s.mu.Lock()
	s.m.Delete(uid)
	s.mu.Unlock()
}

// Destroy очищает все данные
func (a *PointerArena[T]) destroy() {
	for _, s := range a.shards {
		s.mu.Lock()
		s.m = sync.Map{}
		s.mu.Unlock()
	}
	a.counter.Store(0)
}
