package audioplayer

import "sync"

type Subscriptions struct {
	subscriptions     map[string][]func()
	subscriptionsSync sync.Mutex
}

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		subscriptions:     make(map[string][]func()),
		subscriptionsSync: sync.Mutex{},
	}
}

func (s *Subscriptions) Subscribe(key string, f func()) {
	s.subscriptionsSync.Lock()
	defer s.subscriptionsSync.Unlock()

	if s.subscriptions[key] == nil {
		s.subscriptions[key] = make([]func(), 0)
	}
	s.subscriptions[key] = append(s.subscriptions[key], f)
}

func (s *Subscriptions) Emit(key string) {
	s.subscriptionsSync.Lock()
	l, ok := s.subscriptions[key]
	s.subscriptionsSync.Unlock()

	if ok && l != nil {
		go func() {
			for _, f := range l {
				f()
			}
		}()
	}
}
