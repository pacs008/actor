// actorMsg
package actor

import (
	// "fmt"
	log "github.com/sirupsen/logrus"
)

const (
	MsgTypeMessage = iota
	MsgTypeCall
	MsgTypeEvent
)
const (
	MsgTypePoison = iota + 9000
	MsgTypeTimeout
)

// all inter-actor messages have this structure
type ActorMsg interface {
	Type() int
	Data() interface{}
	Sender() *ActorRef
	Reply(interface{}, *ActorRef)
	Wrap(wrapper interface{}, who *ActorRef) ActorMsg
	Unwrap() ActorMsg
	IsPoison() bool
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

// create an ActorMsg
func NewActorMsg(data interface{}, sender *ActorRef) ActorMsg {
	return newActorMsg(MsgTypeMessage, data, sender)
}

func newActorMsg(msgType int, data interface{}, sender *ActorRef) actorMsg {
	return actorMsg{msgType, data, sender, nil}
}

// reply to a message
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

// wrap the message
func (msg actorMsg) Wrap(wrapper interface{}, who *ActorRef) ActorMsg {
	return actorMsg{msg.Type(), wrapper, who, &msg}
}

// unwrap the message
func (msg actorMsg) Unwrap() ActorMsg {
	w := msg.wrapped
	// this code looks crazy, but is needed to avoid typed nil
	if w == nil {
		return nil
	}
	return *w
}

// check poison message
func (msg actorMsg) IsPoison() bool {
	return msg.Type() == MsgTypePoison
}

// check timeout message
func (msg actorMsg) IsTimeout() bool {
	return msg.Type() == MsgTypeTimeout
}
