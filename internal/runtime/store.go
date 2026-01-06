// 包 runtime 提供服务运行时的不可变状态（支持原子切换）。
package runtime

import "sync/atomic"

type Store struct {
	v atomic.Value // *Runtime
}

func NewStore(initial *Runtime) *Store {
	s := &Store{}
	s.v.Store(initial)
	return s
}

func (s *Store) Load() *Runtime {
	v := s.v.Load()
	if v == nil {
		return nil
	}
	return v.(*Runtime)
}

func (s *Store) Store(next *Runtime) {
	s.v.Store(next)
}
