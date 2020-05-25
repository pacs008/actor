// actor_examples_test
package actor

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

func ExampleActorSystemBuilder() {
	// create the actor system
	actorSystem := BuildActorSystem(). // this returns ActorSystemBuilder
						Run()

	// create the actor
	greeter, err := actorSystem.NewActor("greeter", func(actor *Actor, msg ActorMsg) {
		fmt.Printf("Hello %v\n", msg.Data())
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Tom", nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Hello Tom
}

func ExampleActorSystemBuilder_WithSystemData() {
	// create the actor system with system data
	actorSystem := BuildActorSystem().
		WithSystemData("Hello").
		Run()

	// create the actor
	greeter, err := actorSystem.NewActor("greeter", func(actor *Actor, msg ActorMsg) {
		fmt.Printf("%v %v\n", actor.SystemData().(string), msg.Data())
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Tom", nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Hello Tom
}

func ExampleActorSystemBuilder_WithDeadLetterQueue() {
	as := BuildActorSystem().WithDeadLetterQueue(func(_ *Actor, msg ActorMsg) {
		fmt.Printf("DeadLetterQueue received %v", msg.Data())
	}).
		Run()

	as.ToDeadLetter(NewActorMsg("Dead", nil))

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// DeadLetterQueue received Dead
}

func ExampleActorSystemBuilder_WithDeadLetterQueue_panic() {
	actorSystem := BuildActorSystem().WithDeadLetterQueue(func(_ *Actor, msg ActorMsg) {
		fmt.Println(msg.Data())
		msg = msg.Unwrap()
		if msg != nil {
			fmt.Printf("Message that caused panic: %v\n", msg.Data())
		}
	}).
		Run()

	// create the actor
	nervous, err := actorSystem.NewActor("nervous", func(actor *Actor, msg ActorMsg) {
		// force a panic; message will be written to DLQ
		_ = msg.Data().(string)
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the nervous actor
	nervous.Send(1, nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// nervous caught panic: interface conversion: interface {} is int, not string
	// Message that caused panic: 1
}

func ExampleActor() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor
	greeter, err := actorSystem.NewActor("greeter", func(actor *Actor, msg ActorMsg) {
		fmt.Printf("Hello %v\n", msg.Data())
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Tom", nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Hello Tom
}

func ExampleActor_overload() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor
	slow, err := actorSystem.NewActor("slow", func(actor *Actor, msg ActorMsg) {
		// deliberate delay in the actor
		time.Sleep(time.Duration(20 * time.Millisecond))
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// create an actor to monitor the system bus for problems
	actorSystem.BuildActor("SysBusMon", func(ac *Actor, msg ActorMsg) {
		switch msg.Type() {
		case MsgTypeEvent:
			fmt.Printf("SysBusMon: %v\n", msg.Data())
		default:
			fmt.Printf("%v received unexpected type %T\n", ac.Name(), msg)
		}
	}).WithEnter(func(ac *Actor) {
		ac.ActorSystem().SystemBus().Subscribe(ac.Ref(), ActorProblem, nil)
	}).Run()

	// Hammer the actor with messages until it overloads
	for {
		err := slow.Send("", nil)
		if err != nil {
			fmt.Printf("Send failed: %v\n", err)
			break
		}
	}

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Unordered output:
	// Send failed: Send to slow failed: queue full
	// SysBusMon: slow overload
}

func ExampleActor_killed() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor
	victim, err := actorSystem.NewActor("victim", func(actor *Actor, msg ActorMsg) {
		fmt.Printf("Hello %v\n", msg.Data())
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the victim actor
	err = victim.Send("Tom", nil)
	if err != nil {
		fmt.Printf("Failed to send to actor: %v\n", err)
	}

	// kill the actor
	victim.Kill()

	// try sending to the killed actor
	err = victim.Send("Dick", nil)
	if err != nil {
		fmt.Printf("Failed to send to actor: %v\n", err)
	}

	// Prove that a second Kill has no effect
	victim.Kill()

	// wait for result
	time.Sleep(time.Duration(60 * time.Millisecond))

	// Unordered output:
	// Hello Tom
	// Failed to send to actor: Send to victim failed: closed
}

func ExampleActorSystem_Lookup() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor
	_, err := actorSystem.NewActor("greeter", func(actor *Actor, msg ActorMsg) {
		fmt.Printf("Hello %v\n", msg.Data())
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// get the greeter reference from the directory
	greeter, err := actorSystem.Lookup("greeter")

	// check for error
	if err != nil {
		fmt.Printf("Failed to lookup greeter: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Dick", nil)

	// look up non-existent actor
	_, err = actorSystem.Lookup("meeter")

	// check for error
	if err != nil {
		fmt.Printf("Failed to lookup meeter: %v\n", err)
	}

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Unordered output due to async processing

	// Unordered output:
	// Hello Dick
	// Failed to lookup meeter: No actor named [meeter]

}

func ExampleActorSystem_BuildRouter() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actors
	vowel, err := actorSystem.NewActor("vowel", func(actor *Actor, msg ActorMsg) {
		fmt.Printf("%v received %v\n", actor.Name(), msg.Data())
	})
	consonant, err := actorSystem.NewActor("consonant", func(actor *Actor, msg ActorMsg) {
		fmt.Printf("%v received %v\n", actor.Name(), msg.Data())
	})

	// Build the router
	router, err := actorSystem.BuildRouter("router").AddRoute(func(data interface{}) *ActorRef {
		str := data.(string)
		if strings.ContainsAny(str[0:1], "aeiou") {
			return vowel
		}
		return consonant
	}).
		Run()

	// check for error
	if err != nil {
		fmt.Printf("Failed to create router: %v\n", err)
	}

	// send messages to the router
	for _, country := range []string{
		"algeria",
		"bulgaria",
		"eritrea",
		"greece",
		"india",
		"latvia",
	} {
		router.Send(country, nil)
	}

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Unordered output:
	// vowel received algeria
	// consonant received bulgaria
	// vowel received eritrea
	// consonant received greece
	// vowel received india
	// consonant received latvia
}

func ExampleActorSystem_ListActors() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actors
	for _, name := range []string{"Larry", "Curly", "Moe"} {
		_, err := actorSystem.NewActor(name, func(actor *Actor, msg ActorMsg) {
			// do nothing
		})
		// check for error
		if err != nil {
			fmt.Printf("Failed to create actor: %v\n", err)
		}
	}

	for _, name := range actorSystem.ListActors() {
		fmt.Println(name)
	}

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Unordered output:
	// Larry
	// Curly
	// Moe
}

func ExampleActor_closure() {
	// create the actor system
	actorSystem := NewActorSystem()

	// actors should normally be stateless, but if you really
	// need to maintain state you can close over state variables
	f1, f2 := 0, 1
	// create the actor
	fibo, err := actorSystem.NewActor("fibo", func(actor *Actor, msg ActorMsg) {
		f1, f2 = f2, f1+f2
		fmt.Printf("%v\n", f2)
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// generate 5 terms of Fibonacci series
	for i := 0; i < 5; i++ {
		fibo.Send("", nil)
	}

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// 1
	// 2
	// 3
	// 5
	// 8
}

func ExampleActorSystem_SystemBus() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create an actor to monitor the system bus
	// Note: regex to monitor actor lifecycle evens and problems
	actorSystem.BuildActor("SysBusMon", func(ac *Actor, msg ActorMsg) {
		switch msg.Type() {
		case MsgTypeEvent:
			fmt.Printf("%v\n", msg.Data())
		default:
			fmt.Printf("%v received unexpected type %T", ac.Name(), msg)
		}
	}).WithEnter(func(ac *Actor) {
		ac.ActorSystem().SystemBus().Subscribe(ac.Ref(),
			ActorProblem+"|"+ActorLifecycle,
			nil)
	}).Run()

	// create the actor
	test, err := actorSystem.
		BuildActor("test", func(actor *Actor, msg ActorMsg) {
			fmt.Printf("Hello %v\n", msg.Data().(string))
		}).
		WithEnter(func(actor *Actor) {
			// do nothing
		}).
		WithExit(func(actor *Actor) {
			// do nothing
		}).
		Run()

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send an OK message
	test.Send("Tom", nil)
	// force a panic
	test.Send(1, nil)
	// send an nother OK message to prove it's still running
	test.Send("Dick", nil)
	// kill the actor
	test.Kill()

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Unordered output:
	// SysBusMon running
	// test registered
	// test enterFunc
	// test running
	// Hello Tom
	// test caught panic: interface conversion: interface {} is int, not string
	// Hello Dick
	// test exitFunc
	// test unregistered

}

func ExampleActor_After() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor
	greeter, err := actorSystem.NewActor("greeter", func(actor *Actor, msg ActorMsg) {
		if msg.Data().(string) == "Tom" {
			actor.After(time.Duration(250*time.Millisecond), "myself")
		}
		fmt.Printf("Hello %v\n", msg.Data())
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Tom", nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Hello Tom
	// Hello myself
}

func ExampleActor_Every() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor
	var start time.Time     // time the actor started
	var ch chan interface{} // channel to cancel timer
	_, err := actorSystem.
		BuildActor("timer", func(actor *Actor, msg ActorMsg) {
			tenths := time.Since(start).Nanoseconds() / (1000 * 1000 * 100)
			fmt.Printf("Elapsed %v/10 sec\n", tenths)
			if tenths == 5 {
				ch <- "stop"
			}
		}).
		WithEnter(func(actor *Actor) {
			// mark the start time and start the timer. Save the stop channel.
			start = time.Now()
			ch = actor.Every(time.Duration(100*time.Millisecond), "")
		}).
		Run()

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// wait for result
	time.Sleep(time.Duration(550 * time.Millisecond))

	// Output:
	// Elapsed 1/10 sec
	// Elapsed 2/10 sec
	// Elapsed 3/10 sec
	// Elapsed 4/10 sec
	// Elapsed 5/10 sec
}

func ExampleActor_SystemData() {
	type mySystemData struct {
		myString string
		myInt    int
	}
	// create the actor system
	actorSystem := BuildActorSystem().WithSystemData(mySystemData{"Answer", 42}).Run()

	// create the actor
	actor, err := actorSystem.NewActor("actor", func(actor *Actor, msg ActorMsg) {
		msd := actor.SystemData().(mySystemData)
		fmt.Printf("SystemData %v %v\n", msd.myString, msd.myInt)
	})

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// Send message to actor
	actor.Send("", nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// SystemData Answer 42
}

func ExampleActorBuilder() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor using builder - see WithEnter, WithExit etc.
	// to see how the basic actor can be decorated
	greeter, err := actorSystem.
		BuildActor("greeter", func(actor *Actor, msg ActorMsg) {
			fmt.Printf("Hello %v\n", msg.Data())
		}).
		Run() // run is needed to start the actor and return the ACtorRef

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Tom", nil)

	// get the greeter reference from the directory
	greeter1, err := actorSystem.Lookup("greeter")

	// check for error
	if err != nil {
		fmt.Printf("Failed to lookup actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter1.Send("Dick", nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Hello Tom
	// Hello Dick

}

func ExampleActorBuilder_WithEnter() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor using builder
	// add WithEnter to send a message to self after 250 ms
	greeter, err := actorSystem.
		BuildActor("greeter", func(actor *Actor, msg ActorMsg) {
			fmt.Printf("Hello %v\n", msg.Data())
		}).
		WithEnter(func(actor *Actor) {
			actor.After(time.Duration(250*time.Millisecond), "Dick")
		}).
		Run() // run is needed to start the actor and return the ACtorRef

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Tom", nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Hello Tom
	// Hello Dick

}

func ExampleActorBuilder_WithExit() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor using builder
	// add WithExit to say goodbye when killed
	greeter, err := actorSystem.
		BuildActor("greeter", func(actor *Actor, msg ActorMsg) {
			fmt.Printf("Hello %v\n", msg.Data())
		}).
		WithExit(func(actor *Actor) {
			fmt.Printf("Goodbye\n")
		}).
		Run() // run is needed to start the actor and return the ACtorRef

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// send a message to the greeter actor
	greeter.Send("Tom", nil)
	// kill the actor
	greeter.Kill()

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Hello Tom
	// Goodbye

}

func ExampleActorBuilder_WithPool() {
	// create the actor system
	actorSystem := NewActorSystem()

	// create the actor using builder
	// add WithPool to create an actor pool
	greeter, err := actorSystem.
		BuildActor("greeter", func(actor *Actor, msg ActorMsg) {
			// greet and say which pool instance is responding
			fmt.Printf("Hello %v (%v)\n", msg.Data(), actor.Instance())
		}).
		WithPool(3).
		Run() // run is needed to start the actor and return the ActorRef

	// check for error
	if err != nil {
		fmt.Printf("Failed to create actor: %v\n", err)
	}

	// Send 6 messages; each pool member should receive 2
	// but due to asyncronous execution, results may arrive
	// out of order.
	for i := 0; i < 6; i++ {
		greeter.Send("Tom", nil)
	}

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Unordered output:
	// Hello Tom (0)
	// Hello Tom (1)
	// Hello Tom (2)
	// Hello Tom (0)
	// Hello Tom (1)
	// Hello Tom (2)

}

func ExampleActorRef_Call() {
	// create the actor system
	actorSystem := NewActorSystem()

	type callType struct {
		command string
		data    interface{}
	}
	// create the called actor
	called, err := actorSystem.NewActor("called", func(actor *Actor, msg ActorMsg) {
		callRequest := msg.(CallRequest)
		switch callRequest.Method() {
		case "square":
			num, ok := callRequest.Parameters().(int)
			if ok {
				callRequest.CallResponse(num*num, nil)
			} else {
				callRequest.CallResponse(nil, fmt.Errorf("I cannot square %T", callRequest.Parameters()))
			}
		case "cube":
			// error checking omitted here
			num := callRequest.Parameters().(int)
			callRequest.CallResponse(num*num*num, nil)
		case "timeout":
			// do nothing, the caller will receive a timeout
		default:
			callRequest.CallResponse(nil, fmt.Errorf("Unknown command %v", callRequest.Method()))
		}
	})
	// check for error
	if err != nil {
		fmt.Printf("Failed to create called actor: %v\n", err)
	}

	// create the caller actor
	caller, err := actorSystem.NewActor("caller", func(actor *Actor, msg ActorMsg) {
		call := msg.Data().(callType)
		ret, err := called.Call(call.command, call.data, 100)
		fmt.Printf("Call returned %v (err %v)\n", ret, err)
	})
	// check for error
	if err != nil {
		fmt.Printf("Failed to create caller actor: %v\n", err)
	}

	// square an integer to calling actor
	caller.Send(callType{"square", 4}, nil)
	// cube an integer to calling actor
	caller.Send(callType{"cube", 5}, nil)
	// send an unknown command
	caller.Send(callType{"sqrt", 16}, nil)
	// square a bad type
	caller.Send(callType{"square", "7"}, nil)
	// trigger a timeout
	caller.Send(callType{"timeout", ""}, nil)

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Output:
	// Call returned 16 (err <nil>)
	// Call returned 125 (err <nil>)
	// Call returned <nil> (err Unknown command sqrt)
	// Call returned <nil> (err I cannot square string)
	// Call returned <nil> (err Timeout on call to called (100 ms))

}

func ExampleEventBus() {
	as := NewActorSystem()

	// Create event bus, filter to ensure only strings get published
	eb := NewEventBus(func(data interface{}) bool {
		_, ok := data.(string)
		return ok
	})

	// this will throw an error
	err := eb.Publish("bad", 123)
	fmt.Println(err.Error())

	// Demonstrate pattern matching
	for i, pattern := range []string{"africa", "a.*c.*a", "a.*a"} {
		pat := pattern // local copy for this actor
		a, err := as.BuildActor(fmt.Sprintf("Subs %v", i), func(ac *Actor, msg ActorMsg) {
			switch msg.Type() {
			case MsgTypeEvent:
				event := msg.(BusEvent)
				fmt.Printf("%v (%v) received pubSub %v/%v\n", ac.Name(), pat, event.Topic(), event.Data())
			default:
				fmt.Printf("%v (%v) received message %v\n", ac.Name(), pat, msg.Data())
			}
		}).
			WithEnter(func(ac *Actor) {
				err := eb.Subscribe(ac.Ref(), pattern, func(data interface{}) bool {
					food, ok := data.(string)
					return ok && food != "broccoli" // nobody like broccoli
				})
				if err != nil {
					fmt.Printf("%v: Subscribe failed %v\n", ac.Name(), err)
				}
			}).
			Run()
		if err != nil {
			fmt.Println(err.Error())
		}
		// actors can handle normal messages as well as pub/sub
		a.Send("Hello", nil) // just to prove
	}

	for _, data := range []struct {
		country string
		food    string
	}{{"america", "hamburger"},
		{"africa", "ackee"},
		{"asia", "noodles"},
		{"albania", "broccoli"}} {
		fmt.Printf("Publish %v/%v\n", data.country, data.food)
		eb.Publish(data.country, data.food)

	}

	// wait for result
	time.Sleep(time.Duration(500 * time.Millisecond))

	// Unordered output:
	// Wrong message type for bus
	// Subs 0 (africa) received message Hello
	// Subs 1 (a.*c.*a) received message Hello
	// Subs 2 (a.*a) received message Hello
	// Publish america/hamburger
	// Subs 1 (a.*c.*a) received pubSub america/hamburger
	// Subs 2 (a.*a) received pubSub america/hamburger
	// Publish africa/ackee
	// Subs 0 (africa) received pubSub africa/ackee
	// Subs 2 (a.*a) received pubSub africa/ackee
	// Subs 1 (a.*c.*a) received pubSub africa/ackee
	// Publish asia/noodles
	// Subs 2 (a.*a) received pubSub asia/noodles
	// Publish albania/broccoli
}

// ActorMsg is an interface so this example does
// not appear in the godoc documentation :(
func ExampleActorMsg_Wrap() {
	wrapped := "wrapped"
	m := NewActorMsg(wrapped, nil)

	wrapper := "wrapper"
	m = m.Wrap(wrapper, nil)

	fmt.Printf("Wrapper: %v\n", m.Data())
	m = m.Unwrap()
	fmt.Printf("Wrapped: %v\n", m.Data())
	m = m.Unwrap()
	fmt.Printf("Nil: %v\n", m)

	// Output:
	// Wrapper: wrapper
	// Wrapped: wrapped
	// Nil: <nil>
}

func ExampleMux() {
	as := NewActorSystem()

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
				fmt.Printf(ac.Name()+" Timeout '%v'\n", str)
			} else {
				fmt.Printf(ac.Name()+" Sending '%v'\n", str)
				mux.Send(msg.Data(), ac.Ref())
			}
		} else {
			fmt.Printf(ac.Name()+" Received '%v'\n", str)
		}
	}

	aRef, err := as.BuildActor("ActorA", fn).Run()
	if err != nil {
		fmt.Println("Create ActorA failed")
	}

	bRef, err := as.BuildActor("ActorB", fn).Run()
	if err != nil {
		fmt.Println("Create ActorB failed")
	}

	// normal mux messages
	aMsg := []byte("aaa")
	aRef.Send(aMsg, nil)

	aMsg = []byte("bbb")
	bRef.Send(aMsg, nil)

	// test mux timeout
	aMsg = []byte("ccc")
	bRef.Send(aMsg, nil)

	time.Sleep(time.Duration(1200 * time.Millisecond))

	// Unordered output:
	// ActorA Sending 'aaa'
	// ActorA Received 'aaaRsp'
	// ActorB Sending 'bbb'
	// ActorB Sending 'ccc'
	// ActorB Received 'bbbRsp'
	// ActorB Timeout 'ccc'
}

/*
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
*/
