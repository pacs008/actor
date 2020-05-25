package actor

import (
	"fmt"
	"sync"
)

// Topics on the System Message Bus
const (
	ActorLifecycle = "actorLifecycle"
	ActorProblem   = "actorProblem"
)

// The system that all actors operate in.
type ActorSystem struct {
	actors map[string]*ActorRef
	sysBus EventBus
	dlq    *ActorRef
	sync.Mutex
	userData interface{}
}

// Create an actor system.
func NewActorSystem() *ActorSystem {
	return BuildActorSystem().Run()
}

// Register the actor.
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

// Unregister the actor.
func (as *ActorSystem) unregister(name string) {
	as.Lock()
	delete(as.actors, name)
	as.Unlock()

	as.sysBus.Publish(ActorLifecycle, name+" unregistered")
}

// Get an ActorRef by the name of the actor.
func (as *ActorSystem) Lookup(name string) (*ActorRef, error) {
	ref, ok := as.actors[name]
	if !ok {
		return nil, fmt.Errorf("No actor named [%v]", name)
	} else {
		return ref, nil
	}
}

// Return a list of all the actors in the system.
func (as *ActorSystem) ListActors() []string {
	keys := make([]string, len(as.actors))

	i := 0
	for k := range as.actors {
		keys[i] = k
		i++
	}

	return keys
}

// Send an ActorMsg to the DLQ.
func (as *ActorSystem) ToDeadLetter(msg ActorMsg) {
	as.dlq.Forward(msg)
}

// Get the system bus. This is a special bus that publishes
// actor lifecycle events:
// registered
// enterFunc
// running
// exitFunc
// unregistered
// caught panic
func (as *ActorSystem) SystemBus() *EventBus {
	return &as.sysBus
}

// Get the system data.
func (as *ActorSystem) SystemData() interface{} {
	return as.userData
}
