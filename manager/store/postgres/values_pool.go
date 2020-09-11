package postgres

import "sync"

type ValuesPool struct {
	sync.Mutex

	count   int
	fields  int
	allMake chan []interface{}
}

func NewValuesPool(count, fields, size int) *ValuesPool {
	return &ValuesPool{count: count, fields: fields, allMake: make(chan []interface{}, size)}
}

func (vp *ValuesPool) Get() []interface{} {
	vp.Lock()
	defer vp.Unlock()

	select {
	case val := <-vp.allMake:
		return val
	default:
		return make([]interface{}, vp.count*vp.fields)
	}
}

func (vp *ValuesPool) Put(val []interface{}) {
	vp.Lock()
	defer vp.Unlock()

	select {
	case vp.allMake <- val:
	default:
		val = nil
	}
}
