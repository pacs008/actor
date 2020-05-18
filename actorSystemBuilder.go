// actorSystemBuilder
package actor

import (
	log "github.com/sirupsen/logrus"
)

// Builder for actor system.
type ActorSystemBuilder struct {
	as         *ActorSystem
	dlqBuilder *ActorBuilder
}

// Start building an actor system.
func BuildActorSystem() *ActorSystemBuilder {
	as := &ActorSystem{actors: make(map[string]*ActorRef)}
	// create the dead letter queue
	dlqBuilder := as.BuildActor("dlq", func(_ *Actor, msg ActorMsg) {
		name := "<nil>"
		if msg.Sender() != nil {
			name = msg.Sender().name
		}
		log.WithFields(log.Fields{
			"reason": "DLQ",
			"source": name,
		}).Error(msg.Data())
		for true {
			if msg = msg.Unwrap(); msg == nil {
				break
			}
			log.WithFields(log.Fields{
				"reason": "DLQ (wrapped)",
				"source": msg.Sender().name,
			}).Error(msg.Data())
		}
	}).
		withHidden() // hide it

	// create the system event bus
	as.sysBus = NewEventBus(nil)

	return &ActorSystemBuilder{
		as,
		dlqBuilder,
	}
}

// Assign user data to the actor system.
func (sb *ActorSystemBuilder) WithUserData(userData interface{}) *ActorSystemBuilder {
	sb.as.userData = userData
	return sb
}

func (sb *ActorSystemBuilder) WithDeadLetterQueue(dlqFn func(a *Actor, msg ActorMsg)) *ActorSystemBuilder {
	sb.dlqBuilder = sb.as.BuildActor("dlq", dlqFn).withHidden()
	return sb
}

func (sb *ActorSystemBuilder) Run() *ActorSystem {
	dlqRef, err := sb.dlqBuilder.Run()
	if err != nil {
		log.Fatalf("DLQ actor failed to start: %v", err)
	}
	sb.as.dlq = dlqRef
	return sb.as
}
