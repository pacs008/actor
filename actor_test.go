// actor_test
package actor

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

// test message wrapping & unwrapping
func TestMsg(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	wrapped := "wrapped"
	m := NewActorMsg(wrapped, nil)

	wrapper := "wrapper"
	m = m.Wrap(wrapper, nil)

	if m.Data() != wrapper {
		t.Errorf("expected %v, got %v", wrapper, m.Data())
	}
	m = m.Unwrap()
	if m == nil {
		t.Errorf("expected %v, got %v", wrapped, m)
	} else if m.Data() != wrapped {
		t.Errorf("expected %v, got %v", wrapped, m.Data())
	}
	m = m.Unwrap()
	if m != nil {
		t.Errorf("expected nil, got %v", m)
	}
}

// test the dead letter queue
func TestDLQ(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()

	as.ToDeadLetter(NewActorMsg("Dead as a doornail", nil))
}

// test a single actor - get reference by
// creation and by lookup
func TestActor(t *testing.T) {
	type userType struct {
		world string
	}
	log.SetLevel(log.DebugLevel)
	as := BuildActorSystem().WithUserData(&userType{"world"}).Run()
	ch := make(chan string, 0)

	fn := func(ac *Actor, msg ActorMsg) {
		str := msg.Data().(string)
		ch <- str + " " + ac.UserData().(*userType).world
	}

	// check we can create actor
	a, err := as.NewActor("test", fn)
	if err != nil {
		t.Error("Create actor failed")
	}

	// send to actor ref
	a.Send("Hello", nil)
	rsp := <-ch
	log.Infof("Received %v", rsp)

	// lookup and send to it
	a1, err := as.Lookup("test")
	if err != nil {
		t.Error("Lookup actor failed")
	}
	a1.Send("Tata", nil)
	rsp = <-ch
	log.Infof("Received %v", rsp)

	// send a non-string to check panic handling
	a1.Send(1, nil)
	time.Sleep(1 * time.Second)
}

// TestCallError
func TestCallError(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)

	aRsp, err := as.NewActor("aRsp", func(ac *Actor, msg ActorMsg) {
		log.Infof("aRsp got %v (%T)", msg, msg)
		if _, ok := msg.(CallRequest); ok {
			log.Infof("It's a CallRequest")
		} else {
			log.Infof("It's NOT a CallRequest")
		}
		switch msg.(type) {
		case CallRequest:
			msg.(CallRequest).CallResponse("response", fmt.Errorf("Test error return"))
		default:
			log.Errorf("Expected CallRequest but got %v", msg.Type())
		}
	})

	if err != nil {
		t.Error("Create actor aRsp failed")
	}
	aReq, err := as.NewActor("aReq", func(ac *Actor, msg ActorMsg) {
		rsp, err := aRsp.Call("myMethod", msg.Data(), 1000)
		log.Infof("TestCallError %v", err)
		switch rsp.(type) {
		case string:
			ch <- rsp.(string)
		default:
			ch <- fmt.Sprintf("Unexpected type: %T", rsp)
		}
	})
	if err != nil {
		t.Error("Create actor aReq failed")
	}

	// Make requester send
	aReq.Send("request", nil)
	log.Infof("Sent request")
	rsp := <-ch
	if rsp != "response" {
		t.Errorf("Expected 'response' got '%v'", rsp)
	}
	log.Infof("Received '%v'", rsp)
}

// test a pair of actors, one forwards to the
// other, which replies to the first
func TestReqRsp(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)

	// create a pair of actors
	fnRsp := func(ac *Actor, msg ActorMsg) {
		msg.Reply("response", nil)
	}

	aRsp, err := as.NewActor("aRsp", fnRsp)
	if err != nil {
		t.Error("Create actor aRsp failed")
	}
	fnReq := func(ac *Actor, msg ActorMsg) {
		str := msg.Data().(string)
		switch str {
		case "request":
			aRsp.Send(msg.Data(), ac.Ref())
		case "response":
			ch <- str
		}
	}

	aReq, err := as.NewActor("aReq", fnReq)
	if err != nil {
		t.Error("Create actor aReq failed")
	}

	// send to actor ref
	aReq.Send("request", nil)
	rsp := <-ch
	if rsp != "response" {
		t.Errorf("Expected 'response' got '%v'", rsp)
	}
	log.Infof("Received '%v'", rsp)

}

// test an actor with After
func TestAfter(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)

	// create an actor with after
	doFunc := makeChanWriterFn(ch)
	enterFunc := func(ac *Actor) {
		ac.After(time.Duration(1*time.Second), "after")
	}
	as.BuildActor("aAfter", doFunc).WithEnter(enterFunc).Run()
	// _, err := as.NewActor("aAfter", doFunc, enterFunc)
	// if err != nil {
	// 	t.Error("Create actor aAfter failed")
	// }

	rsp := <-ch
	log.Infof("Received '%v'", rsp)

}

// test an actor with Every
func TestEvery(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)

	// create an actor with every
	doFunc := makeChanWriterFn(ch)
	enterFunc := func(ac *Actor) {
		ac.Every(time.Second, "every")
	}
	_, err := as.BuildActor("aEvery", doFunc).WithEnter(enterFunc).Run()
	if err != nil {
		t.Error("Create actor aEvery failed")
	}

	rsp := <-ch
	log.Infof("Received '%v'", rsp)
	rsp = <-ch
	log.Infof("Received '%v'", rsp)

}

// test event bus
func TestEventBus(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()

	eb := NewEventBus(func(data interface{}) bool {
		_, ok := data.(string)
		return ok
	})

	// check that system bus works
	/*_, err := */
	as.BuildActor("SysBusMon", func(ac *Actor, msg ActorMsg) {
		switch msg.Type() {
		case MsgTypeEvent:
			log.Infof("%v received BusEvent %v", ac.Name(), msg.Data())
		default:
			log.Errorf("%v received unexpected type %T", ac.Name(), msg)
		}
	}).WithEnter(func(ac *Actor) {
		ac.ActorSystem().SystemBus().Subscribe(ac.Ref(), ActorLifecycle, nil)
	}).Run()

	// check that topic matching works
	for _, pattern := range []string{"topic", "^t.*", "^[r-u].*", "^[a-c].*"} {
		a, err := as.BuildActor(fmt.Sprintf("subscriber %v", pattern), func(ac *Actor, msg ActorMsg) {
			switch msg.Type() {
			case MsgTypeEvent:
				event := msg.(BusEvent)
				log.Infof("%v received pubSub %v/%v", ac.Name(), event.Topic(), event.Data())
			default:
				log.Infof("%v received message %v", ac.Name(), msg.Data())
			}
		}).
			WithEnter(func(ac *Actor) {
				err := eb.Subscribe(ac.Ref(), pattern, nil)
				if err != nil {
					log.Errorf("%v: Subscribe failed %v", ac.Name(), err)
				}
			}).Run()
		if err != nil {
			t.Error(err.Error())
		}
		a.Send("Hello", nil)
	}
	eb.Publish("topic", fmt.Sprintf("Some event"))

	// create three actors, each subscribing with a particular filter
	subscribers := make([]*ActorRef, 0)
	for i := 0; i < 3; i++ {
		myStr := strconv.Itoa(i)
		a, err := as.BuildActor(fmt.Sprintf("subscriber%v", i), func(ac *Actor, msg ActorMsg) {
			switch msg.Type() {
			case MsgTypeEvent:
				event := msg.(BusEvent)
				log.Infof("%v received %v", ac.Name(), event.Data())
			}
		}).
			WithEnter(func(ac *Actor) {
				eb.Subscribe(ac.Ref(), "", func(data interface{}) bool {
					if s, ok := data.(string); ok && strings.Index(s, myStr) > -1 {
						return true
					}
					return false
				})
			}).Run()
		if err != nil {
			t.Error(err.Error())
		}
		subscribers = append(subscribers, a)
	}

	for i := 0; i < 3; i++ {
		eb.Publish("myTopic", fmt.Sprintf("Event #%v", i))
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	// kill them to check lifecycle
	for _, subscriber := range subscribers {
		subscriber.Kill()
	}
	time.Sleep(time.Duration(500) * time.Millisecond)
}

// test round robin pool
func TestRobin(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)

	// create an actor with every
	doFunc := func(ac *Actor, msg ActorMsg) {
		str := msg.Data().(string)
		ch <- str + fmt.Sprintf("%v", ac.Instance())
	}

	aRobin, err := as.BuildActor("aRobin", doFunc).WithPool(10).Run()
	if err != nil {
		t.Error("Create actor aRobin failed")
	}

	for i := 0; i < 20; i++ {
		aRobin.Send("Hello ", nil)
		rsp := <-ch
		log.Infof("Received '%v'", rsp)
	}
	aRobin.Kill()
	time.Sleep(time.Second)
}

// test router
// use example rather than test just for a change!
func ExampleRouter() {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)
	// check that we can make an actor loop function
	// by closing over local variables
	makeWriter := func(ch chan string, rsp string) func(*Actor, ActorMsg) {
		return func(_ *Actor, msg ActorMsg) {
			ch <- rsp
		}
	}

	a1, err := as.NewActor("a1", makeWriter(ch, "a1"))
	if err != nil {
		log.Error("Create actor a1 failed")
	}
	a2, err := as.NewActor("a2", makeWriter(ch, "a2"))
	if err != nil {
		log.Error("Create actor a2 failed")
	}
	ar, err := as.NewRouter("r1", func(d interface{}) *ActorRef {
		switch {
		case strings.Contains(d.(string), "a1"):
			return a1
		case strings.Contains(d.(string), "a2"):
			return a2
		}
		return nil
	})
	if err != nil {
		log.Error("Create router failed")
	}

	ar.Send("This is a1", nil)
	fmt.Printf(<-ch + "\n")
	ar.Send("This is a2", nil)
	fmt.Printf(<-ch + "\n")

	// Output:
	// a1
	// a2
}

// test mux
func TestMux(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()

	// Init function acts like a mock
	// Copy request channel to response, appending
	// "Rsp" to the message. If the message is "ccc"
	// then delay for 5 seconds before replying
	muxInitFn := func(outCh chan<- []byte) chan []byte {
		inCh := make(chan []byte, 0) // Mux reads this channel
		go func() {
			log.Debug("muxFn starting")
			expect := []string{"aaa", "bbb", "ccc"}
			i := 0
			for msg := range inCh {
				if 0 != bytes.Compare(msg, []byte(expect[i])) {
					t.Errorf("Expected %v got %v", expect[i], msg)
				}
				if i == 2 {
					time.Sleep(2 * time.Second)
				}
				// log.Debug("muxFn relaying " + string(msg))
				inCh <- append(msg, []byte("Rsp")...)

				i++
			}
		}()
		return inCh
	}

	// create a multiplexer that sends to chan
	mux, err := as.BuildMux("aMux", muxInitFn).
		WithKeyBuilder(func(s interface{}) string { return string(s.([]byte))[0:3] }).
		WithTimeout(1000).
		Run()
	fn := makeActorSendBytesFn(mux)

	aRef, err := as.BuildActor("ActorA", fn).
		WithExit(func(ac *Actor) { log.Info(ac.Name() + " shuffling off its mortal coil") }).
		Run()
	if err != nil {
		t.Error("Create ActorA failed")
	}

	bRef, err := as.BuildActor("ActorB", fn).Run()
	if err != nil {
		t.Error("Create ActorB failed")
	}

	// see what actors we have ...
	// log.Infof("Actors in system %v", as.ListActors())

	// normal mux messages
	aMsg := []byte("aaa")
	aRef.Send(aMsg, nil)
	time.Sleep(1 * time.Second)

	aMsg = []byte("bbb")
	bRef.Send(aMsg, nil)

	// test mux timeout
	aMsg = []byte("ccc")
	bRef.Send(aMsg, nil)

	// try sending a poison message
	time.Sleep(2 * time.Second)
	aRef.Kill()
	mux.Kill()
	log.Debug("sent poison")
	time.Sleep(2 * time.Second)
}

// check that we can make an actor loop function
// by closing over local variables
func makeChanWriterFn(ch chan string) func(*Actor, ActorMsg) {
	return func(_ *Actor, msg ActorMsg) {
		str := msg.Data().(string)
		ch <- str
	}
}

func makeActorSenderFn(ref *ActorRef) func(*Actor, ActorMsg) {
	return func(ac *Actor, msg ActorMsg) {
		str := msg.Data().(string)
		if len(str) == 3 {
			if msg.IsTimeout() {
				log.Infof(ac.Name()+" Timeout '%v'", str)
			} else {
				log.Infof(ac.Name()+" Sending '%v'", str)
				ref.Send(str, ac.Ref())
			}
		} else {
			log.Infof(ac.Name()+" Received '%v'", str)
		}
	}
}

func makeActorSendBytesFn(ref *ActorRef) func(*Actor, ActorMsg) {
	return func(ac *Actor, msg ActorMsg) {
		str := string(msg.Data().([]byte))
		if len(str) == 3 {
			if msg.IsTimeout() {
				log.Infof(ac.Name()+" Timeout '%v'", str)
			} else {
				log.Infof(ac.Name()+" Sending '%v'", str)
				ref.Send(msg.Data(), ac.Ref())
			}
		} else {
			log.Infof(ac.Name()+" Received '%v'", str)
		}
	}
}
