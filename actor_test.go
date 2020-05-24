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
	ch := make(chan ActorMsg, 0)

	as := BuildActorSystem().WithDeadLetterQueue(func(_ *Actor, m ActorMsg) {
		ch <- m
	}).
		Run()

	deadMsg := "Dead"
	as.ToDeadLetter(NewActorMsg(deadMsg, nil))

	msg := <-ch

	data := msg.(ActorMsg).Data()

	if data != deadMsg {
		t.Errorf("DLQ expected %v got %v", deadMsg, data)
	}
}

// test a single actor - get reference by
// creation and by lookup
func TestActor(t *testing.T) {
	type userType struct {
		world string
	}
	log.SetLevel(log.DebugLevel)
	ch := make(chan string, 100)
	busCh := make(chan string, 100)
	dlq := make(chan ActorMsg, 0)
	as := BuildActorSystem().
		WithSystemData(&userType{"world"}).
		WithDeadLetterQueue(func(_ *Actor, msg ActorMsg) {
			dlq <- msg
		}).
		Run()

	fn := func(ac *Actor, msg ActorMsg) {
		str := msg.Data().(string)
		ch <- str + " " + ac.SystemData().(*userType).world
	}

	monitorSysBus(as, busCh)

	// check we can create actor
	a, err := as.NewActor("test", fn)
	if err != nil {
		t.Error("Create actor failed")
	}

	// send to actor ref
	a.Send("Hello", nil)
	rsp := <-ch
	if rsp != "Hello world" {
		t.Errorf("Expected %v got %v", "Hello world", rsp)
	}

	// lookup and send to it
	a1, err := as.Lookup("test")
	if err != nil {
		t.Error("Lookup actor failed")
	}
	a1.Send("Goodbye cruel", nil)
	rsp = <-ch
	if rsp != "Goodbye cruel world" {
		t.Errorf("Expected %v got %v", "Goodbye cruel world", rsp)
	}

	// send a non-string to check panic handling
	a1.Send(999, nil)
	msg := <-dlq
	data := msg.(ActorMsg).Data()
	if data != 999 {
		t.Errorf("DLQ expected %v got %v", 999, data)
	}
	a1.Kill()
	for _, s := range []string{
		"SysBusMon running",
		"test registered",
		"test running",
		"test caught panic: interface conversion: interface {} is int, not string",
	} {
		msg := <-busCh
		if msg != s {
			t.Errorf("Expected %v, got %v", s, msg)
		}
	}
}

// TestCallError
func TestCallError(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)

	aRsp, err := as.NewActor("aRsp", func(ac *Actor, msg ActorMsg) {
		switch msg.(type) {
		case CallRequest:
			callRequest := msg.(CallRequest)
			switch callRequest.Parameters().(string) {
			case "ok":
				callRequest.CallResponse("success", nil)
			case "nok":
				callRequest.CallResponse("fail", fmt.Errorf("Error message"))
			case "timeout":
				// don't reply
			}
		default:
			log.Errorf("Expected CallRequest but got %v", msg.Type())
		}
	})

	if err != nil {
		t.Error("Create actor aRsp failed")
	}
	aReq, err := as.NewActor("aReq", func(ac *Actor, msg ActorMsg) {
		rsp, err := aRsp.Call("myMethod", msg.Data(), 1000)
		// log.Infof("TestCallError %v", err)
		if err != nil {
			ch <- "Error: " + err.Error()
			return
		}
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

	// Send a normal message
	aReq.Send("ok", nil)
	rsp := <-ch
	if rsp != "success" {
		t.Errorf("Expected 'success' got '%v'", rsp)
	}

	// Trigger an error
	aReq.Send("nok", nil)
	rsp = <-ch
	if rsp != "Error: Error message" {
		t.Errorf("Expected 'Error: Error message' got '%v'", rsp)
	}

	// Trigger a timeout
	aReq.Send("timeout", nil)
	rsp = <-ch
	if !strings.HasPrefix(rsp, "Error: Timeout") {
		t.Errorf("Expected 'Error: Timeout*' got '%v'", rsp)
	}
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
	// log.Infof("Received '%v'", rsp)

}

// test an actor with After
func TestAfter(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 0)

	// create an actor with after
	doFunc := makeChanWriterFn(ch)
	enterFunc := func(ac *Actor) {
		go func() {
			<-time.After(time.Duration(500 * time.Millisecond))
			ch <- "after"
		}()
		ac.After(time.Duration(1*time.Second), "sendMeAfter")
	}
	as.BuildActor("aAfter", doFunc).WithEnter(enterFunc).Run()
	// _, err := as.NewActor("aAfter", doFunc, enterFunc)
	// if err != nil {
	// 	t.Error("Create actor aAfter failed")
	// }

	rsp := <-ch
	if rsp != "after" {
		t.Errorf("Expected 'after' got '%v'", rsp)
	}
	rsp = <-ch
	if rsp != "sendMeAfter" {
		t.Errorf("Expected 'sendMeAfter' got '%v'", rsp)
	}
	// log.Infof("Received '%v'", rsp)

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

	for i := 0; i < 2; i++ {
		rsp := <-ch
		if rsp != "every" {
			t.Errorf("Expected 'every' got '%v'", rsp)
		}
	}
}

// example event bus
func ExampleEventBus() {
	log.SetLevel(log.DebugLevel)
	ch := make(chan string, 100)
	as := NewActorSystem()

	eb := NewEventBus(func(data interface{}) bool {
		_, ok := data.(string)
		return ok
	})

	// check that system bus works
	monitorSysBus(as, ch)

	// check that topic matching works
	for _, pattern := range []string{"topic", "^t.*", "^[r-u].*", "^[a-c].*"} {
		a, err := as.BuildActor(fmt.Sprintf("subscriber %v", pattern), func(ac *Actor, msg ActorMsg) {
			switch msg.Type() {
			case MsgTypeEvent:
				event := msg.(BusEvent)
				ch <- fmt.Sprintf("%v received pubSub %v/%v", ac.Name(), event.Topic(), event.Data())
			default:
				ch <- fmt.Sprintf("%v received message %v", ac.Name(), msg.Data())
			}
		}).
			WithEnter(func(ac *Actor) {
				err := eb.Subscribe(ac.Ref(), pattern, nil)
				if err != nil {
					ch <- fmt.Sprintf("%v: Subscribe failed %v", ac.Name(), err)
				}
			}).Run()
		if err != nil {
			ch <- err.Error()
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
				ch <- fmt.Sprintf("%v received %v", ac.Name(), event.Data())
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
			ch <- err.Error()
		}
		subscribers = append(subscribers, a)
	}

	for i := 0; i < 3; i++ {
		eb.Publish("myTopic", fmt.Sprintf("Event #%v", i))
	}

	// kill them to check lifecycle
	for _, subscriber := range subscribers {
		subscriber.Kill()
	}

	timer := time.NewTimer(time.Duration(500) * time.Millisecond)

	for done := false; !done; {
		select {
		case msg := <-ch:
			fmt.Println(msg)
		case <-timer.C:
			done = true
		}
	}

	// Unordered output:
	// SysBusMon running
	// subscriber topic registered
	// subscriber topic enterFunc
	// subscriber topic running
	// subscriber ^t.* registered
	// subscriber ^t.* enterFunc
	// subscriber ^t.* running
	// subscriber ^t.* received message Hello
	// subscriber ^t.* received pubSub topic/Some event
	// subscriber1 received Event #1
	// subscriber topic received message Hello
	// subscriber topic received pubSub topic/Some event
	// subscriber ^[r-u].* received message Hello
	// subscriber0 received Event #0
	// subscriber ^[r-u].* received pubSub topic/Some event
	// subscriber ^[r-u].* registered
	// subscriber ^[r-u].* enterFunc
	// subscriber ^[r-u].* running
	// subscriber ^[a-c].* received message Hello
	// subscriber2 received Event #2
	// subscriber ^[a-c].* registered
	// subscriber ^[a-c].* enterFunc
	// subscriber ^[a-c].* running
	// subscriber0 registered
	// subscriber0 enterFunc
	// subscriber0 running
	// subscriber1 registered
	// subscriber1 enterFunc
	// subscriber1 running
	// subscriber2 registered
	// subscriber2 enterFunc
	// subscriber2 running
	// subscriber2 unregistered
	// subscriber1 unregistered
	// subscriber0 unregistered
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
		// rsp := <-ch
		// log.Infof("Received '%v'", rsp)
	}

	// add the instance numbers 0..9 * 2 = 90
	sum := 0
	n := 0
	for i := 0; i < 20; i++ {
		rsp := <-ch
		fmt.Sscanf(rsp, "Hello %d", &n)
		sum += n
		// log.Infof("Received '%v'", rsp)
	}
	if sum != 90 {
		t.Errorf("Expected 90 got %v", sum)
	}
	aRobin.Kill()
	// time.Sleep(time.Second)
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
func ExampleMux() {
	log.SetLevel(log.DebugLevel)
	as := NewActorSystem()
	ch := make(chan string, 100)

	// Init function acts like a mock
	// Copy request channel to response, appending
	// "Rsp" to the message. If the message is "ccc
	// then delay for 2 seconds before replying
	muxInitFn := func(toResChan <-chan []byte) chan []byte {
		fromResChan := make(chan []byte, 0) // Mux reads this channel
		go func() {
			// log.Debug("muxFn starting")
			for msg := range toResChan {
				if 0 == bytes.Compare(msg, []byte("ccc")) {
					go func() {
						<-time.After(time.Duration(1000 * time.Millisecond))
						fromResChan <- append(msg, []byte("Rsp")...)
					}()
					continue
				}
				//log.Debug("muxFn relaying " + string(msg))
				fromResChan <- append(msg, []byte("Rsp")...)
			}
		}()
		return fromResChan
	}

	// create a multiplexer that sends to chan
	mux, err := as.BuildMux("aMux", muxInitFn).
		WithKeyBuilder(func(s interface{}) string { return string(s.([]byte))[0:3] }).
		WithTimeout(500).
		Run()

	// Simple function for test actors
	fn := func(ac *Actor, msg ActorMsg) {
		str := string(msg.Data().([]byte))
		if len(str) == 3 {
			if msg.IsTimeout() {
				ch <- fmt.Sprintf(ac.Name()+" Timeout '%v'", str)
			} else {
				ch <- fmt.Sprintf(ac.Name()+" Sending '%v'", str)
				mux.Send(msg.Data(), ac.Ref())
			}
		} else {
			ch <- fmt.Sprintf(ac.Name()+" Received '%v'", str)
		}
	}

	aRef, err := as.BuildActor("ActorA", fn).
		WithExit(func(ac *Actor) { log.Info(ac.Name() + " shuffling off its mortal coil") }).
		Run()
	if err != nil {
		log.Error("Create ActorA failed")
	}

	bRef, err := as.BuildActor("ActorB", fn).Run()
	if err != nil {
		log.Error("Create ActorB failed")
	}

	// see what actors we have ...
	// log.Infof("Actors in system %v", as.ListActors())

	// normal mux messages
	aMsg := []byte("aaa")
	aRef.Send(aMsg, nil)

	aMsg = []byte("bbb")
	bRef.Send(aMsg, nil)

	// test mux timeout
	aMsg = []byte("ccc")
	bRef.Send(aMsg, nil)

	timer := time.NewTimer(time.Duration(2000) * time.Millisecond)

	for timeout := false; !timeout; {
		select {
		case msg := <-ch:
			fmt.Printf(msg + "\n")
		case <-timer.C:
			timeout = true
		}
	}
	time.Sleep(2 * time.Second)

	// Unordered output:
	// ActorA Sending 'aaa'
	// ActorA Received 'aaaRsp'
	// ActorB Sending 'bbb'
	// ActorB Sending 'ccc'
	// ActorB Received 'bbbRsp'
	// ActorB Timeout 'ccc'
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

func monitorSysBus(as *ActorSystem, ch chan string) {
	as.BuildActor("SysBusMon", func(ac *Actor, msg ActorMsg) {
		switch msg.Type() {
		case MsgTypeEvent:
			ch <- msg.Data().(string)
		default:
			ch <- fmt.Sprintf("%v received unexpected type %T", ac.Name(), msg)
		}
	}).WithEnter(func(ac *Actor) {
		ac.ActorSystem().SystemBus().Subscribe(ac.Ref(), ActorLifecycle, nil)
	}).Run()
}
