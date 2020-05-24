package actor

import (
	// "fmt"
	log "github.com/sirupsen/logrus"
)

// Normal message types
const (
	MsgTypeMessage = iota
	MsgTypeCall
	MsgTypeEvent
)

// System message types
const (
	MsgTypePoison = iota + 9000
	MsgTypeTimeout
)

// All inter-actor messages implement this interface.
type ActorMsg interface {
	// The type of the message - MsgTypeMessage etc.
	Type() int

	// The payload of the message - can be any data type.
	Data() interface{}

	// The sender of the message - may be nil.
	Sender() *ActorRef

	// Send a reply to the sender of the message. If
	// the message's Sender is nil, the reply goes
	// to the dead letter queue.
	Reply(interface{}, *ActorRef)

	// Wraps the message as payload inside another message.
	Wrap(wrapper interface{}, who *ActorRef) ActorMsg

	// Returns the wrapped message - nil if no wrapped message.
	Unwrap() ActorMsg

	// Special message type used to kill actors.
	// This message type is never seen passed to the actor.
	IsPoison() bool

	// Special message type used to indicate a timeout.
	IsTimeout() bool
}

type actorMsg struct {
	msgType int
	data    interface{}
	sender  *ActorRef
	wrapped *actorMsg
}

// Type method
func (am actorMsg) Type() int {
	return am.msgType
}

// Data method
func (am actorMsg) Data() interface{} {
	return am.data
}

// Sender method
func (am actorMsg) Sender() *ActorRef {
	return am.sender
}

// Create an ActorMsg.
func NewActorMsg(data interface{}, sender *ActorRef) ActorMsg {
	return newActorMsg(MsgTypeMessage, data, sender)
}

func newActorMsg(msgType int, data interface{}, sender *ActorRef) actorMsg {
	return actorMsg{msgType, data, sender, nil}
}

// Reply to a message. If the sender of the message is nil, the
// message is dropped.
func (msg actorMsg) Reply(data interface{}, replyTo *ActorRef) {
	// TODO should send to DLQ if IsNoreply but we don't have ActorSystem
	// if msg.sender.IsNoreply() {
	// 	SEND TO DLQ
	// }
	if msg.sender == nil {
		log.Errorf("Cannot reply to message with nil sender (%v)", msg)
		return
	}
	msg.sender.Send(data, replyTo)
}

// Wrap the message.
func (msg actorMsg) Wrap(wrapper interface{}, who *ActorRef) ActorMsg {
	return actorMsg{msg.Type(), wrapper, who, &msg}
}

// Unwrap the message.
func (msg actorMsg) Unwrap() ActorMsg {
	w := msg.wrapped
	// this code looks crazy, but is needed to avoid typed nil
	if w == nil {
		return nil
	}
	return *w
}

// Private method to check poison message.
func (msg actorMsg) IsPoison() bool {
	return msg.Type() == MsgTypePoison
}

// Check timeout message.
func (msg actorMsg) IsTimeout() bool {
	return msg.Type() == MsgTypeTimeout
}
