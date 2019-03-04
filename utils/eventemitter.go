// https://flaviocopes.com/golang-event-listeners/

package utils

import (
	"sync"
)

type EventEmitter struct {
	listenersMap map[string][]chan interface{}
	mux          sync.Mutex
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		listenersMap: map[string][]chan interface{}{},
	}
}

// AddListener adds an event listener to the Dog struct instance
func (emitter *EventEmitter) AddListener(e string, ch chan interface{}) {
	emitter.mux.Lock()
	if _, ok := emitter.listenersMap[e]; ok {
		emitter.listenersMap[e] = append(emitter.listenersMap[e], ch)
	} else {
		emitter.listenersMap[e] = []chan interface{}{ch}
	}
	emitter.mux.Unlock()
}

// RemoveListener removes an event listener from the Dog struct instance
func (emitter *EventEmitter) RemoveListener(e string, ch chan interface{}) {
	emitter.mux.Lock()
	if _, ok := emitter.listenersMap[e]; ok {
		for i := range emitter.listenersMap[e] {
			if emitter.listenersMap[e][i] == ch {
				emitter.listenersMap[e] = append(emitter.listenersMap[e][:i], emitter.listenersMap[e][i+1:]...)
				break
			}
		}
	}
	emitter.mux.Unlock()
}

// Emit emits an event on the Dog struct instance
func (emitter *EventEmitter) Emit(e string, data interface{}) {
	emitter.mux.Lock()
	listeners, ok := emitter.listenersMap[e]
	emitter.mux.Unlock()
	if !ok {
		return
	}
	for _, handler := range listeners {
		go func(handler chan interface{}) {
			handler <- data
		}(handler)
	}
}
