package actor

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// Actor is the core of the actor package. It is
// created by an ActorBuilder. An actor does not have
// any methods accessible from outside; it can only
// be accessed by passing messages to its ActorRef.
//
// An Actor runs in its own goroutine. It processes
// messages that it receives in its mailbox by
// calling the message handling function. It can
// communicate with other actors by sending messages
// or calling them. References of the actors to
// communicate with can be obtained by name from the
// actor system directory.
type Actor struct {
	as        *ActorSystem
	mailbox   chan ActorMsg
	doFunc    func(*Actor, ActorMsg)
	enterFunc func(*Actor)
	exitFunc  func(*Actor)
	loopFunc  func(*Actor)
	name      string
	instance  int
	hidden    bool
}

// Create an actor in the system. This is a
// convenience method to create an actor
// without calling ActorBuilder.
func (as *ActorSystem) NewActor(name string, doFunc func(*Actor, ActorMsg)) (*ActorRef, error) {
	return as.BuildActor(name, doFunc).Run()
}

// This is the main loop that reads messages from the
// actor mailbox and invokes the message handler.
// We can't make this a method because we need to use
// it in constructor.
// If the message is a poison message it is intercepted,
// the exit function is called and the actor terminates.
func mainLoop(a *Actor) {
	// read messages forever and pass
	// to the doFunction
	// log.Debugf("%v entering main loop", a.Name())
	for msg := range a.mailbox {
		// check for poison
		if msg.IsPoison() { //msg.Sender().IsPoison() {
			// log.Info(a.Name() + " swallowed poison - exiting")
			if a.exitFunc != nil {
				a.ActorSystem().sysBus.Publish(ActorLifecycle, a.Name()+" exitFunc")
				a.exitFunc(a)
			}
			a.as.unregister(a.name)
			return
		}

		protect(a, msg, a.doFunc)
	}
}

// Function to handle panics thrown by an actor. The message
// that was being handled is written to the Dead Letter Queue
// and the actor continues with processing the next message.
// Note: if the DLQ also panics (which should not be possible),
// the actor dies.
func protect(a *Actor, m ActorMsg, doFunc func(a *Actor, m ActorMsg)) {
	defer func() {
		// log.Debug("protect checking recover") // Println executes normally even if there is a panic
		if x := recover(); x != nil {
			// log.Debugf("%v caught panic: %v", a.Name(), x)
			a.ActorSystem().sysBus.Publish(ActorLifecycle, fmt.Sprintf("%v caught panic: %v", a.Name(), x))

			if a.Name() == "dlq" {
				log.Fatalf("Urgh! DLQ loop - really dying")
			}
			a.ActorSystem().ToDeadLetter(m) // would be good to wrap this with a message
		}
	}()
	// log.Debug("protect calling doFunc")
	doFunc(a, m)
	// log.Debug("protect returned doFunc")
}

// run the actor
func (a Actor) run(as *ActorSystem) (*ActorRef, error) {
	if !a.hidden {
		err := as.register(a.Ref())
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}

	if a.enterFunc != nil {
		a.ActorSystem().sysBus.Publish(ActorLifecycle, a.Name()+" enterFunc")
		// log.Debugf("Calling %v.enterFunc", a.name)
		a.enterFunc(&a)
	}
	// log.Debugf("Running %v.mainLoop", a.name)
	as.sysBus.Publish(ActorLifecycle, a.name+" running")
	go a.loopFunc(&a) // mainLoop(&a)

	return a.Ref(), nil
}

// check valid name
func (a Actor) validName() error {
	var err error = nil
	name := a.name
	if name == "" || name[0] == '!' {
		err = fmt.Errorf("Invalid Actor name " + name)
	}
	return err
}

// Get the ActorRef for this actor - used to set Sender in messages.
func (a *Actor) Ref() *ActorRef {
	return &ActorRef{a.ActorSystem(), &a.mailbox, a.name}
}

// Get the name for this actor.
func (a *Actor) Name() string {
	return a.name
}

// Get the instance number for this actor (used when
// the actor is a member of a pool).
func (a *Actor) Instance() int {
	return a.instance
}

// Send self a message after the specified duration. This
// fires one-off.
func (a *Actor) After(d time.Duration, data interface{}) {
	go func() {
		<-time.After(d)
		a.mailbox <- NewActorMsg(data, nil)
	}()
}

// Send self a message every specified duration. This
// fires repeatedly. It returns a channel - write anything
// to this channel to stop the timer.
func (a *Actor) Every(d time.Duration, data interface{}) chan interface{} {
	ch := make(chan interface{}, 0)
	go func() {
		ticker := time.NewTicker(d)
		for {
			select {
			case <-ticker.C:
				a.mailbox <- NewActorMsg(data, nil)
			case <-ch:
				ticker.Stop()
			}
		}
	}()
	return ch
}

// Get the ActorSystem in which the actor is running.
func (a *Actor) ActorSystem() *ActorSystem {
	return a.as
}

// Get the SystemData for the ActorSystem in which the actor is running.
func (a *Actor) SystemData() interface{} {
	return a.as.SystemData()
}
