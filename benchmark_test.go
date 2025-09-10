package alchemist_test

import (
	"alchemist"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

type KeyModel struct {
	Id int
}
type ValueModel struct {
	Val string
}

type GoMutexMap struct {
	mu   sync.Mutex
	data map[int]*ValueModel
}

func NewGoMutexMap() *GoMutexMap               { return &GoMutexMap{data: make(map[int]*ValueModel)} }
func (m *GoMutexMap) Set(k int, v *ValueModel) { m.mu.Lock(); m.data[k] = v; m.mu.Unlock() }
func (m *GoMutexMap) Get(k int) *ValueModel    { m.mu.Lock(); v := m.data[k]; m.mu.Unlock(); return v }
func (m *GoMutexMap) Del(k int)                { m.mu.Lock(); delete(m.data, k); m.mu.Unlock() }

type GoRWMutexMap struct {
	mu   sync.RWMutex
	data map[int]*ValueModel
}

func NewGoRWMutexMap() *GoRWMutexMap             { return &GoRWMutexMap{data: make(map[int]*ValueModel)} }
func (m *GoRWMutexMap) Set(k int, v *ValueModel) { m.mu.Lock(); m.data[k] = v; m.mu.Unlock() }
func (m *GoRWMutexMap) Get(k int) *ValueModel    { m.mu.RLock(); v := m.data[k]; m.mu.RUnlock(); return v }
func (m *GoRWMutexMap) Del(k int)                { m.mu.Lock(); delete(m.data, k); m.mu.Unlock() }

// --- Bench: Go map + Mutex
func BenchmarkGoMutexMap(b *testing.B) {
	m := NewGoMutexMap()
	var c int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(atomic.AddInt64(&c, 1))
			k := i
			v := &ValueModel{Val: fmt.Sprintf("val-%d", i)}
			m.Set(k, v)
			_ = m.Get(k)
			m.Del(k)
		}
	})
}

// --- Bench: Go map + RWMutex
func BenchmarkGoRWMutexMap(b *testing.B) {
	m := NewGoRWMutexMap()
	var c int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(atomic.AddInt64(&c, 1))
			k := i
			v := &ValueModel{Val: fmt.Sprintf("val-%d", i)}
			m.Set(k, v)
			_ = m.Get(k)
			m.Del(k)
		}
	})
}

// --- Bench: AlchemistMap
func BenchmarkAlchemistMap(b *testing.B) {
	m := alchemist.NewAlchemistMap[KeyModel, ValueModel]()
	defer m.Destroy()
	var counter int64
	ops := int64(10_000_000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt64(&counter, 1)
			if i > ops {
				return
			}
			k := alchemist.NewAlchemistValue(&KeyModel{Id: int(i)})
			v := alchemist.NewAlchemistValue(&ValueModel{Val: fmt.Sprintf("val-%d", i)})
			m.Set(k, v)
			_ = m.Get(k)
			m.Remove(k)
		}
	})
}

func BenchmarkAlchemistMapBatchOperations(b *testing.B) {
	const N = 1_000_000 // количество элементов

	m := alchemist.NewAlchemistMap[KeyModel, ValueModel]()
	defer m.Destroy()

	// создаём ключи и значения один раз
	keys := make([]*alchemist.AlchemistValue[KeyModel], N)
	values := make([]*alchemist.AlchemistValue[ValueModel], N)
	uids := make([]uintptr, N) // для хранения UID ключей
	for i := 0; i < N; i++ {
		k := &KeyModel{Id: i}
		v := &ValueModel{Val: "val-" + strconv.Itoa(i)}
		keys[i] = alchemist.NewAlchemistValue(k)
		values[i] = alchemist.NewAlchemistValue(v)
		uids[i] = keys[i].GetUIDValue() // сразу сохраняем UID
	}

	b.ResetTimer()

	b.Run("BatchSet", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.BatchSet(keys, values)
		}
	})

	b.Run("BatchGet", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := m.BatchGet(keys)
			for j, v := range got {
				if v == nil {
					b.Fatalf("missing value at index %d", j)
				}
			}
		}
	})

	b.Run("BatchRemove", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// используем сохранённые UID и PointerArena для получения ключей
			keysToRemove := make([]*alchemist.AlchemistValue[KeyModel], N)
			for j := 0; j < N; j++ {
				keysToRemove[j] = m.Keys.Get(uids[j])
			}

			removed := m.BatchRemove(keysToRemove)
			for j, v := range removed {
				if v == nil {
					b.Fatalf("missing value at index %d during remove", j)
				}
			}
		}
	})
}
