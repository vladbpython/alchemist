package alchemist

import "sync"

var AlchemistValuePool = sync.Pool{
	New: func() any {
		return &AlchemistValue[any]{}
	},
}

type AlchemistValue[T any] struct {
	uid     uintptr
	pointer *T
}

func NewAlchemistValue[T any](pointer *T) *AlchemistValue[T] {
	return &AlchemistValue[T]{
		pointer: pointer,
		uid:     0, // по умолчанию 0, будет задан при Alloc в арене
	}
}

func (v *AlchemistValue[T]) setUIDValue(uid uintptr) {
	v.uid = uid
}

func (v *AlchemistValue[T]) GetUIDValue() uintptr {
	return v.uid
}

func (v *AlchemistValue[T]) Value() *T {
	return v.pointer
}
