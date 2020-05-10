// eventBus
package actor

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

type BusEvent interface {
	ActorMsg
	Timestamp() time.Time
	Topic() string
}

type busEvent struct {
	ActorMsg
	timestamp time.Time
	topic     string
}

// internal book-keeping
type subscriber struct {
	actorRef *ActorRef
	regexp   *regexp.Regexp
	filter   func(interface{}) bool
}

// event bus - not much to it really!
type eventBus struct {
	sync.Mutex
	filter      func(interface{}) bool
	subscribers []subscriber
}

func NewBusEvent(topic string, msg interface{}, caller *ActorRef) BusEvent {
	return busEvent{newActorMsg(MsgTypeEvent, msg, caller), time.Now(), topic}
}

func NewEventBus(filter func(interface{}) bool) eventBus {
	return eventBus{filter: filter, subscribers: make([]subscriber, 0)}
}

// subscribe an actor to the event bus
func (bus *eventBus) Subscribe(ar *ActorRef, pattern string, filter func(interface{}) bool) error {
	bus.Lock()
	defer bus.Unlock()
	found := false
	for _, subs := range bus.subscribers {
		if ar == subs.actorRef {
			found = true
			break
		}
	}
	if !found {
		var rx *regexp.Regexp
		if pattern != "" {
			var err error
			rx, err = regexp.Compile(pattern)
			if err != nil {
				return err
			}
		}
		bus.subscribers = append(bus.subscribers, subscriber{ar, rx, filter})
	}
	return nil
}

// unsubscribe the actor from the event bus
func (bus *eventBus) Unsubscribe(ar *ActorRef) {
	bus.Lock()
	defer bus.Unlock()
	found := false
	idx := -1
	var subs subscriber
	for idx, subs = range bus.subscribers {
		if ar == subs.actorRef {
			found = true
			break
		}
	}
	if found {
		bus.subscribers = append(bus.subscribers[:idx], bus.subscribers[idx+1:]...)
	}
}

// publish to all subscribers
func (bus *eventBus) Publish(topic string, msg interface{}) error {
	be := NewBusEvent(topic, msg, nil)
	if bus.filter != nil && !bus.filter(msg) {
		return fmt.Errorf("Wrong message type for bus")
	}
	for _, subs := range bus.subscribers {
		if (subs.regexp == nil || subs.regexp.MatchString(topic)) &&
			(subs.filter == nil || subs.filter(msg)) {
			subs.actorRef.SendMsg(be)
		}
	}
	return nil
}

func (be busEvent) Timestamp() time.Time {
	return be.timestamp
}

func (be busEvent) Topic() string {
	return be.topic
}
