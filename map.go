package alchemist

/*
#cgo darwin LDFLAGS: -L${SRCDIR} -lalchemist_c
#cgo linux  LDFLAGS: -L${SRCDIR} -lalchemist_c
#cgo windows LDFLAGS: -L${SRCDIR} -lalchemist_c
#include <stdint.h>
#include <stddef.h>

typedef struct AlchemistMap AlchemistMap;
typedef struct AlchemistMapIterator AlchemistMapIterator;

// alias для uintptr_t
typedef uintptr_t alch_uintptr_t;

// функции работы с map
extern AlchemistMap* alchemist_map_new();
extern void alchemist_map_destroy(AlchemistMap* obj);
extern void alchemist_map_set(AlchemistMap* obj, alch_uintptr_t key, alch_uintptr_t value);
extern alch_uintptr_t alchemist_map_get(AlchemistMap* obj, alch_uintptr_t key);
extern alch_uintptr_t alchemist_map_remove(AlchemistMap* obj, alch_uintptr_t key);

// батчевые операции
extern void alchemist_map_batch_set(AlchemistMap* obj, const alch_uintptr_t* keys, const alch_uintptr_t* values, size_t len);
extern void alchemist_map_batch_get(AlchemistMap* obj, const alch_uintptr_t* keys, alch_uintptr_t* values_out, size_t len);
extern void alchemist_map_batch_remove(AlchemistMap* obj, const alch_uintptr_t* keys, alch_uintptr_t* values_out, size_t len);

// функции итератора
extern AlchemistMapIterator* alchemist_map_iterator_new(AlchemistMap* obj);
extern void alchemist_map_iterator_destroy(AlchemistMapIterator* it);
extern int alchemist_map_iterator_next(AlchemistMapIterator* it, alch_uintptr_t* key, alch_uintptr_t* value);
extern size_t alchemist_map_iterator_next_batch(AlchemistMapIterator* it, alch_uintptr_t* keys_out, alch_uintptr_t* vals_out, size_t max_len);
*/
import "C"
import (
	"unsafe"
)

// AlchemistMap Go-обёртка
type AlchemistMap[K any, V any] struct {
	obj  *C.AlchemistMap
	Keys *PointerArena[K]
	Vals *PointerArena[V]
}

func NewAlchemistMap[K any, V any]() *AlchemistMap[K, V] {
	return &AlchemistMap[K, V]{
		obj:  C.alchemist_map_new(),
		Keys: NewPointerArena[K](),
		Vals: NewPointerArena[V](),
	}
}

func (m *AlchemistMap[K, V]) Destroy() {
	if m.obj != nil {
		C.alchemist_map_destroy(m.obj)
		m.obj = nil
	}
	m.Keys.destroy()
	m.Vals.destroy()
}

func (m *AlchemistMap[K, V]) Set(k *AlchemistValue[K], v *AlchemistValue[V]) {
	keyUID := m.Keys.alloc(k)
	valUID := m.Vals.alloc(v)
	C.alchemist_map_set(m.obj, C.alch_uintptr_t(keyUID), C.alch_uintptr_t(valUID))
}

func (m *AlchemistMap[K, V]) Get(k *AlchemistValue[K]) *AlchemistValue[V] {
	valUID := C.alchemist_map_get(m.obj, C.alch_uintptr_t(k.GetUIDValue()))
	if valUID == 0 {
		return nil
	}
	return m.Vals.Get(uintptr(valUID))
}

func (m *AlchemistMap[K, V]) Iter() chan struct {
	Key   *AlchemistValue[K]
	Value *AlchemistValue[V]
} {
	ch := make(chan struct {
		Key   *AlchemistValue[K]
		Value *AlchemistValue[V]
	})
	go func() {
		iter := C.alchemist_map_iterator_new(m.obj)
		if iter == nil {
			close(ch)
			return
		}
		defer C.alchemist_map_iterator_destroy(iter)

		var keyUID, valUID C.alch_uintptr_t
		for C.alchemist_map_iterator_next(iter, &keyUID, &valUID) != 0 {
			k := m.Keys.Get(uintptr(keyUID))
			v := m.Vals.Get(uintptr(valUID))
			if k == nil || v == nil {
				continue // пропускаем nil-значения
			}
			ch <- struct {
				Key   *AlchemistValue[K]
				Value *AlchemistValue[V]
			}{Key: k, Value: v}
		}
		close(ch)
	}()
	return ch
}

func (m *AlchemistMap[K, V]) Remove(k *AlchemistValue[K]) {
	valUID := C.alchemist_map_remove(m.obj, C.alch_uintptr_t(k.GetUIDValue()))
	if valUID != 0 {
		m.Keys.free(k.GetUIDValue())
		m.Vals.free(uintptr(valUID))
	}
}

// BatchSet вставляет сразу несколько ключей и значений
func (m *AlchemistMap[K, V]) BatchSet(keys []*AlchemistValue[K], vals []*AlchemistValue[V]) {
	n := len(keys)
	if n == 0 {
		return
	}

	// Подготавливаем массивы uintptr заранее
	keyUIDs := make([]uintptr, n)
	valUIDs := make([]uintptr, n)

	for i := 0; i < n; i++ {
		// Получаем AlchemistValue из пула, если нужно
		k := keys[i]
		v := vals[i]

		keyUIDs[i] = m.Keys.alloc(k)
		valUIDs[i] = m.Vals.alloc(v)
	}
	C.alchemist_map_batch_set(
		m.obj,
		(*C.uintptr_t)(unsafe.Pointer(&keyUIDs[0])),
		(*C.uintptr_t)(unsafe.Pointer(&valUIDs[0])),
		C.size_t(n),
	)
}

func (m *AlchemistMap[K, V]) BatchGet(keys []*AlchemistValue[K]) []*AlchemistValue[V] {
	n := len(keys)
	if n == 0 {
		return nil
	}

	uids := make([]uintptr, n)
	for i, k := range keys {
		if k != nil {
			uids[i] = k.GetUIDValue() // <- берем существующий UID
		}
	}

	vals := make([]uintptr, n)
	C.alchemist_map_batch_get(
		m.obj,
		(*C.uintptr_t)(unsafe.Pointer(&uids[0])),
		(*C.uintptr_t)(unsafe.Pointer(&vals[0])),
		C.size_t(n),
	)

	var out []*AlchemistValue[V]
	for _, v := range vals {
		if v != 0 {
			out = append(out, m.Vals.Get(uintptr(v)))
		}
	}
	return out
}

func (m *AlchemistMap[K, V]) BatchRemove(keys []*AlchemistValue[K]) []*AlchemistValue[V] {
	n := len(keys)
	if n == 0 {
		return nil
	}

	uids := make([]uintptr, n)
	for i, k := range keys {
		if k != nil {
			uids[i] = k.GetUIDValue() // <- берем существующий UID
		}
	}

	vals := make([]uintptr, n)
	C.alchemist_map_batch_remove(
		m.obj,
		(*C.uintptr_t)(unsafe.Pointer(&uids[0])),
		(*C.uintptr_t)(unsafe.Pointer(&vals[0])),
		C.size_t(n),
	)

	var out []*AlchemistValue[V]
	for _, v := range vals {
		if v != 0 {
			out = append(out, m.Vals.Get(uintptr(v)))
		}
	}
	return out
}

// Итератор
type AlchemistMapIterator struct {
	obj  *C.AlchemistMapIterator
	keys []uintptr
	vals []uintptr
	pos  int
}

func (m *AlchemistMap[K, V]) Iterator() *AlchemistMapIterator {
	it := C.alchemist_map_iterator_new(m.obj)
	if it == nil {
		return nil
	}
	return &AlchemistMapIterator{obj: it}
}

func (it *AlchemistMapIterator) Next() (uintptr, uintptr, bool) {
	var k, v C.alch_uintptr_t
	ret := C.alchemist_map_iterator_next(it.obj, &k, &v)
	if ret == 0 {
		return 0, 0, false
	}
	return uintptr(k), uintptr(v), true
}

func (it *AlchemistMapIterator) NextBatch(max int) ([]uintptr, []uintptr) {
	if it.obj == nil || max == 0 {
		return nil, nil
	}

	keys := make([]C.alch_uintptr_t, max)
	vals := make([]C.alch_uintptr_t, max)

	n := C.alchemist_map_iterator_next_batch(it.obj, &keys[0], &vals[0], C.size_t(max))
	if n == 0 {
		return nil, nil
	}

	outKeys := make([]uintptr, n)
	outVals := make([]uintptr, n)
	for i := 0; i < int(n); i++ {
		outKeys[i] = uintptr(keys[i])
		outVals[i] = uintptr(vals[i])
	}

	return outKeys, outVals
}

func (it *AlchemistMapIterator) Destroy() {
	if it.obj != nil {
		C.alchemist_map_iterator_destroy(it.obj)
		it.obj = nil
	}
}
