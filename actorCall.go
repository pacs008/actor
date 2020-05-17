package actor

import (
	"fmt"
	"time"
	// log "github.com/sirupsen/logrus"
)

// CallRequest is a special type of ActorMsg used
// to make a synchronous call to another actor.
// The receiving actor must use the message's
// CallResponse method to reply.
type CallRequest struct {
	callRequest
}

type callRequest struct {
	ActorMsg
	method    string
	replyChan chan interface{}
}

type callResponse struct {
	payload interface{}
	err     error
}

// Call makes a synchronous call to another actor by sending a
// CallRequest message to to it. The receiving actor must reply
// by calling the CallRequest.CallResponse method.
func (ref *ActorRef) Call(method string,
	req interface{},
	timeoutMs int) (interface{}, error) {
	reqMsg := CallRequest{
		callRequest{newActorMsg(MsgTypeCall, req, nil),
			method,
			make(chan interface{})}}
	ref.SendMsg(reqMsg)
	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer close(reqMsg.replyChan)
	defer timer.Stop()
	select {
	case rsp := <-reqMsg.replyChan:
		// if it's structured as a callResponse, pass the data and error
		if data, ok := rsp.(callResponse); ok {
			return data.payload, data.err
		}
		// OK it's some unstructured stuff
		return rsp, nil
	case <-timer.C:
		return nil, fmt.Errorf("Call to %v timed out (%v ms)", ref.name, timeoutMs)
	}
}

// The method for the call. It is the responsibility
// of the receiving actor to dispatch the call
// appropriately.
func (req CallRequest) Method() string {
	return req.method
}

// Parameters is a synonym for ActorMsg.Data
func (req CallRequest) Parameters() interface{} {
	return req.Data()
}

// CallResponse method must be called to respond
// to the call.
func (req CallRequest) CallResponse(data interface{}, err error) {
	// TODO should send to DLQ if IsNoreply but we don't have ActorSystem
	// if msg.sender.IsNoreply() {
	// 	SEND TO DLQ
	// }
	req.replyChan <- callResponse{data, err}
}
