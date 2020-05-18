package actor

import (
	"fmt"
	"sync"
)

// Topics on the System Message Bus
const (
	ActorLifecycle = "actor_lifecycle"
)

// the parent of all actors
type ActorSystem struct {
	actors map[string]*ActorRef
	sysBus EventBus
	dlq    *ActorRef
	sync.Mutex
	userData interface{}
}

// create an actor system
func NewActorSystem() *ActorSystem {
	return BuildActorSystem().Run()
}

// register the actor
func (as *ActorSystem) register(ar *ActorRef) error {
	as.Lock()
	_, ok := as.actors[ar.name]
	if ok {
		return fmt.Errorf("Actor %v already registered", ar.name)
	}
	as.actors[ar.name] = ar
	as.Unlock()

	as.sysBus.Publish(ActorLifecycle, ar.name+" registered")

	return nil
}

// unregister the actor
func (as *ActorSystem) unregister(name string) {
	as.Lock()
	delete(as.actors, name)
	as.Unlock()

	as.sysBus.Publish(ActorLifecycle, name+" unregistered")
}

// get an actor ref by name
func (as *ActorSystem) Lookup(name string) (*ActorRef, error) {
	ref, ok := as.actors[name]
	if !ok {
		return nil, fmt.Errorf("No actor named [%v]", name)
	} else {
		return ref, nil
	}
}

// return a list of all the actors in the system
func (as *ActorSystem) ListActors() []string {
	keys := make([]string, len(as.actors))

	i := 0
	for k := range as.actors {
		keys[i] = k
		i++
	}

	return keys
}

// send to DLQ
func (as *ActorSystem) ToDeadLetter(msg ActorMsg) {
	as.dlq.Forward(msg)
}

// get the system bus
func (as *ActorSystem) SystemBus() *EventBus {
	return &as.sysBus
}

// get the user data
func (as *ActorSystem) UserData() interface{} {
	return as.userData
}
