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

// EventBus is a pub-sub mechanism. Actors can subscribe
// to topics on the bus. Actors receive events like standard
// ActorMsg, with Type() of MsgTypeEvent.
type EventBus struct {
	eventBus
}

// event bus - not much to it really!
type eventBus struct {
	sync.Mutex
	filter      func(interface{}) bool
	subscribers []subscriber
}

// NewBusEvent creates a new BusEvent.
func NewBusEvent(topic string, msg interface{}, caller *ActorRef) BusEvent {
	return busEvent{
		newActorMsg(MsgTypeEvent, msg, caller),
		time.Now(),
		topic,
	}
}

// NewEventBus creates a new EventBus. If specified, the filter
// function is applied to data of events sent on the bus. If the
// filter returns false, the send fails.
func NewEventBus(filter func(interface{}) bool) EventBus {
	return EventBus{eventBus{filter: filter, subscribers: make([]subscriber, 0)}}
}

// Subscribe allows an actor to subscribe to events on the event bus.
// The pattern is a regex - the actor will only receive events on topics
// that match the regex. In addition, the optional filter is applied
// to event data. If the filter returns false, the actor does not receive it.
func (bus *EventBus) Subscribe(ar *ActorRef, pattern string, filter func(interface{}) bool) error {
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

// Unsubscribe the actor from the event bus.
func (bus *EventBus) Unsubscribe(ar *ActorRef) {
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

// Publish an event to all subscribers.
func (bus *EventBus) Publish(topic string, msg interface{}) error {
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

// Timestamp returns the time at which the event was created.
func (be busEvent) Timestamp() time.Time {
	return be.timestamp
}

// Topic returns the topic of the event.
func (be busEvent) Topic() string {
	return be.topic
}
