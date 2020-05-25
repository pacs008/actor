package actor

import (
	"fmt"
	"sync"
)

// ActorRef is a handle for other actors to send messages
// to the referenced actor.
type ActorRef struct {
	as      *ActorSystem
	mailbox *chan ActorMsg
	name    string
	closed  bool
	sync.Mutex
}

// Name of the referenced actor.
func (ref *ActorRef) Name() string {
	return ref.name
}

// SendMsg sends an ActorMsg to an actor. Same as Forward.
// Note: SendMsg is non-blocking. Check for error return.
// This allows for back-pressure load control.
func (ref *ActorRef) SendMsg(msg ActorMsg) error {
	if ref.closed {
		return fmt.Errorf("Send to %v failed: closed", ref.Name())
	}
	select {
	case *ref.mailbox <- msg:
		return nil
	default:
		ref.as.sysBus.Publish(ActorProblem, ref.name+" overload")
		return fmt.Errorf("Send to %v failed: queue full", ref.Name())
	}
}

// Send is a convenience method to build ActorMsg and send it.
// Note: Send is non-blocking. Check for error return.
// This allows for back-pressure load control.
func (ref *ActorRef) Send(data interface{}, sender *ActorRef) error {
	return ref.SendMsg(NewActorMsg(data, sender))
}

// Forward an ActorMsg to an actor - passes the
// Sender of the original message unchanged.
// Note: Forward is non-blocking. Check for error return.
// This allows for back-pressure load control.
func (ref *ActorRef) Forward(msg ActorMsg) error {
	return ref.SendMsg(msg)
}

// Kill an actor. Only allow kill once - subsequent attempts
// will have no effect.
func (ref *ActorRef) Kill() {
	ref.Lock()
	defer ref.Unlock()
	if ref.closed {
		return
	}
	ref.closed = true
	*ref.mailbox <- newActorMsg(MsgTypePoison, "", nil)
	close(*ref.mailbox)
}
