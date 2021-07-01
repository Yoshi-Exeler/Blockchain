package gorx

import "sync"

type Observable struct {
	mutex         *sync.Mutex
	subscriptions []*Subscription
}

type Subscription struct {
	observable *Observable
	cb         func(v interface{})
}

func (o *Observable) Subscribe(cb func(v interface{})) *Subscription {
	// Make sure to acquire the observable lock
	o.mutex.Lock()
	defer o.mutex.Unlock()
	// create a new subscriber and append it
	sub := &Subscription{observable: o, cb: cb}
	o.subscriptions = append(o.subscriptions, sub)
	return sub
}

func (o *Observable) Push(v interface{}) {
	// Make sure to acquire the observable lock
	o.mutex.Lock()
	defer o.mutex.Unlock()
	// Notify the subscribers of our new value
	for _, sub := range o.subscriptions {
		sub.cb(v)
	}
}

func (s *Subscription) Unsubscribe() {
	// Acquire the lock of the relevant observable
	s.observable.mutex.Lock()
	defer s.observable.mutex.Unlock()
	// Remove this subscription from the subscriber array
	for i := 0; i < len(s.observable.subscriptions); i++ {
		// If this is our subscription
		if s.observable.subscriptions[i] == s {
			// remove this element
			s.observable.subscriptions = remove(s.observable.subscriptions, i)
		}
	}
}

func remove(s []*Subscription, i int) []*Subscription {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}
