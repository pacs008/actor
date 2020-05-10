// actorRef
package actor

// "fmt"

// handle for other actors to send messages
type ActorRef struct {
	as      *ActorSystem
	mailbox *chan ActorMsg
	name    string
}

const (
	noreply = "!noreply"
)

// send a message to an actor
func (ref *ActorRef) SendMsg(msg ActorMsg) {
	*ref.mailbox <- msg
}

// convenience method to build ActorMsg and send
func (ref *ActorRef) Send(data interface{}, sender *ActorRef) {
	if sender == nil {
		sender = NoreplyActorRef()
	}
	*ref.mailbox <- NewActorMsg(data, sender)
}

// forward a message to an actor
func (ref *ActorRef) Forward(msg ActorMsg) {
	*ref.mailbox <- msg
}

// kill an actor
func (ref *ActorRef) Kill() {
	*ref.mailbox <- newActorMsg(MsgTypePoison, "", nil)
}

// Noreply ActorRef
func NoreplyActorRef() *ActorRef {
	return makeSomething(noreply)
}

// is it the timeout actor?
func (ref *ActorRef) IsNoreply() bool {
	return isSomething(ref, noreply)
}

// Timeout ActorRef
// func TimeoutActorRef() *ActorRef {
// 	return makeSomething(timeout)
// }

// is it the timeout actor?
// func (ref *ActorRef) IsTimeout() bool {
// 	return isSomething(ref, timeout)
// }

// make a thing
func makeSomething(something string) *ActorRef {
	return &ActorRef{
		nil,
		nil,
		something,
	}
}

// is it a thing?
func isSomething(ref *ActorRef, something string) bool {
	return ref != nil && ref.name == something
}
