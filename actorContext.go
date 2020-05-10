// actorContext
package actor

import (
	// "fmt"
	"time"
)

// interface for callback functions to use
type ActorContext interface {
	Name() string
	Self() *ActorRef
	Instance() int
	After(time.Duration, interface{})
	Every(time.Duration, interface{})
	ActorSystem() *ActorSystem
}
