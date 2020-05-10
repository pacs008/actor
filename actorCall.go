// actorCall
package actor

import (
	"fmt"
	"time"
	// log "github.com/sirupsen/logrus"
)

type CallRequest interface {
	ActorMsg
	Method() string
	Parameters() interface{}
	CallResponse(data interface{}, err error)
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

func (ar *ActorRef) Call(method string, req interface{}, timeoutMs int) (interface{}, error) {
	reqMsg := callRequest{newActorMsg(MsgTypeCall, req, nil), method, make(chan interface{})}
	ar.SendMsg(reqMsg)
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
		return nil, fmt.Errorf("Call to %v timed out (%v ms)", ar.name, timeoutMs)
	}
}

// call method is the data; parameters are the wrapped data
func (req callRequest) Method() string {
	return req.method
}
func (req callRequest) Parameters() interface{} {
	return req.Data()
}

// reply to a Call
func (msg callRequest) CallResponse(data interface{}, err error) {
	// TODO should send to DLQ if IsNoreply but we don't have ActorSystem
	// if msg.sender.IsNoreply() {
	// 	SEND TO DLQ
	// }
	msg.replyChan <- callResponse{data, err}
}
