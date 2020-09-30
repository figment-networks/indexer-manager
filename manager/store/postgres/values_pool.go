package postgres

import "sync"

// ValuesPool is Pool of fixed size slices of interface{}
type ValuesPool struct {
	sync.Mutex

	count   int
	fields  int
	allMake chan []interface{}
}

// NewValuesPool is ValuesPool constructor
func NewValuesPool(count, fields, size int) *ValuesPool {
	return &ValuesPool{count: count, fields: fields, allMake: make(chan []interface{}, size)}
}

// Get from pool or create new one
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

// Put back to the pool or discard
func (vp *ValuesPool) Put(val []interface{}) {
	vp.Lock()
	defer vp.Unlock()

	select {
	case vp.allMake <- val:
	default:
		val = nil
	}
}
