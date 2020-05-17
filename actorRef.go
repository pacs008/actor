package actor

// "fmt"

// handle for other actors to send messages
type ActorRef struct {
	as      *ActorSystem
	mailbox *chan ActorMsg
	name    string
}

// send a message to an actor
func (ref *ActorRef) SendMsg(msg ActorMsg) {
	*ref.mailbox <- msg
}

// convenience method to build ActorMsg and send
func (ref *ActorRef) Send(data interface{}, sender *ActorRef) {
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
