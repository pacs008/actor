// actor project actor.go
package actor

import (
	"fmt"
	// "log"
	"time"

	log "github.com/sirupsen/logrus"
)

// basic actor
type Actor struct {
	as        *ActorSystem
	mailbox   chan ActorMsg
	doFunc    func(ActorContext, ActorMsg)
	enterFunc func(ActorContext)
	exitFunc  func(ActorContext)
	loopFunc  func(a *Actor)
	name      string
	instance  int
	hidden    bool
}

// create an actor in the system
// convenience method without calling builder
func (as *ActorSystem) NewActor(name string, doFunc func(ActorContext, ActorMsg)) (*ActorRef, error) {
	return as.BuildActor(name, doFunc).Run()
}

// can't make this a method because we need to use it in constructor
func mainLoop(a *Actor) {
	// read messages forever and pass
	// to the doFunction
	// log.Debugf("%v entering main loop", a.Name())
	for msg := range a.mailbox {
		// check for poison
		if msg.IsPoison() { //msg.Sender().IsPoison() {
			// log.Info(a.Name() + " swallowed poison - exiting")
			a.exitFunc(a)
			a.as.unregister(a.name)
			return
		}

		protect(a, msg, a.doFunc)
	}
}

// handle panics
func protect(a ActorContext, m ActorMsg, doFunc func(a ActorContext, m ActorMsg)) {
	defer func() {
		// log.Debug("protect checking recover") // Println executes normally even if there is a panic
		if x := recover(); x != nil {
			log.Debugf("%v caught panic: %v", a.Name(), x)
			if a.Name() == "dlq" {
				log.Fatalf("Urgh! DLQ loop -= really dying")
			}
			a.ActorSystem().ToDeadLetter(m) // would be good to wrap this with a message
		}
	}()
	// log.Debug("protect calling doFunc")
	doFunc(a, m)
	// log.Debug("protect returned doFunc")
}

// get the handle for this actor
func (a *Actor) Self() *ActorRef {
	return &ActorRef{a.ActorSystem(), &a.mailbox, a.name}
}

// get the name for this actor
func (a *Actor) Name() string {
	return a.name
}

func (a *Actor) Instance() int {
	return a.instance
}

// send self a message after specified duration
func (a *Actor) After(d time.Duration, data interface{}) {
	go func() {
		<-time.After(d)
		a.mailbox <- NewActorMsg(data, nil)
	}()
}

// send self a message every specified duration
func (a *Actor) Every(d time.Duration, data interface{}) {
	go func() {
		ticker := time.Tick(d)
		for {
			<-ticker
			a.mailbox <- NewActorMsg(data, nil)
		}
	}()
}

// get the ActorSystem
func (a Actor) ActorSystem() *ActorSystem {
	return a.as
}

// run the actor
func (a Actor) run(as *ActorSystem) (*ActorRef, error) {
	if !a.hidden {
		err := as.register(a.Self())
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}

	if a.enterFunc != nil {
		// log.Debugf("Calling %v.enterFunc", a.name)
		a.enterFunc(&a)
	}
	// log.Debugf("Running %v.mainLoop", a.name)
	as.sysBus.Publish(ACTOR_LIFECYCLE, a.name+" running")
	go a.loopFunc(&a) // mainLoop(&a)

	return a.Self(), nil
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

// clone an actor with a new name
func (a Actor) Clone(name string) Actor {
	b := a
	b.name = name
	return b
}
