package actor

// mux
// An actor that receives requests from multiple callers.
// It stores the payload of the request along with the
// time to live and the sender reference,
// and a correlation ID extracted fron the request message.
// It then forwards the request payload to the destination.
// When a response is received the mux extracts the correlation
// ID, matches it to a stored request, and sends the response
// back to the sender. If there is no matching message, the
// the unmatched message is written to the Dead Letter Queue (?)
// If a stored message times out, it is written to the Dead
// Letter Queue.
//
// This version takes an initializer function that takes an inout
// channel and returns an output channel
//

import (
	// "fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// Messages pending response
type cacheEntry struct {
	ttl  time.Time // time to expire message
	data ActorMsg  // payload
}

// main mux structure
type Mux struct {
	Actor
	cache            map[string]cacheEntry // cache of pending messages
	initFunc         func(chan<- []byte) chan []byte
	kb               func(interface{}) string // function to build key
	reqPassthru      func(interface{}) bool   // function to pass thru req without cache
	rspPassthru      func(interface{}) bool   // function to pass thru rsp without cache
	passthruRspActor ActorRef                 // actor to passthru responses to
	reqChan          chan<- []byte            // channel to pass requests to underlying resource
	rspChan          <-chan []byte            // channel to receive responses from underlying resource
	tick             <-chan time.Time
	timeout          int
}

// override the Actor main loop
func (mux *Mux) muxLoop() {
	// purge cache once per second
	mux.tick = time.Tick(1000 * time.Millisecond)
	// log.Debugf("%v starting loop", mux.Name())
	for {
		if muxProtect(mux.muxDo) {
			break
		}
	}
}

func (mux *Mux) muxDo() bool {
	select {
	case <-mux.tick:
		// log.Debug(mux.Name() + " tick")
		mux.purge()
	case msg := <-mux.mailbox:
		// check for poison
		if msg.IsPoison() { //msg.Sender().IsPoison() {
			log.Info(mux.Name() + " swallowed poison - exiting")
			return true
		}
		// log.Debug(mux.Name() + " read mailbox " + string(msg.data.([]byte)))
		if !mux.reqPassthru(msg.Data()) { // if not passthru then cache
			key := mux.kb(msg.Data())
			mux.cache[key] = cacheEntry{
				time.Now().Add(time.Duration(mux.timeout) * time.Millisecond),
				msg,
			}
		}
		// log.Debug(mux.Name() + " writing reqChannel " + string(msg.data.([]byte)))
		mux.reqChan <- msg.Data().([]byte)
		// log.Debug(mux.Name() + " wrote reqChannel " + string(msg.data.([]byte)))
	// a message back from the embedded resource
	case rsp := <-mux.rspChan:
		log.Debug(mux.Name() + " read rspMailbox " + string(rsp))
		if mux.reqPassthru(rsp) { // if passthru then just send
			mux.passthruRspActor.Send(rsp, nil)
		} else { // if not passthru then look in cache
			key := mux.kb(rsp)
			if cache, ok := mux.cache[key]; ok {
				delete(mux.cache, key)
				log.Debug(mux.Name() + " Replying " + string(rsp))
				cache.data.Reply(rsp, nil)
			} else {
				log.Infof("Key %v not found - send to DLQ", key)
				mux.as.ToDeadLetter(NewActorMsg(rsp, mux.Ref()))
			}
		}
	}
	return false
}

//
func (mux *Mux) purge() {
	now := time.Now()
	for key, cache := range mux.cache {
		if now.After(cache.ttl) {
			// cache.data.Reply(cache.data.Data(), TimeoutActorRef())
			cache.data.Sender().SendMsg(actorMsg{MsgTypeTimeout,
				cache.data.Data(),
				nil, nil})
			delete(mux.cache, key)
		}
	}
}

// run the mux
func (mux Mux) run(as *ActorSystem) (*ActorRef, error) {
	err := as.register(mux.Ref())
	if err != nil {
		log.Error(err)
		return nil, err
	}

	mux.rspChan = mux.initFunc(mux.reqChan)
	go mux.muxLoop() // mainLoop(&a)

	return mux.Ref(), nil
}

// handle panics
func muxProtect(muxDo func() bool) (finished bool) {
	defer func() {
		// log.Debug("protect checking recover") // Println executes normally even if there is a panic
		if x := recover(); x != nil {
			log.Debugf("run time panic: %v", x)
			finished = true
			return
		}
	}()
	// log.Debug("protect calling doFunc")
	muxDo()
	// log.Debug("protec[t returned doFunc")
	finished = false
	return
}
