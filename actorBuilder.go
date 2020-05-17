package actor

import (
	"fmt"
	// "log"
)

// ActorBuilder is used to build and
// decorate actors. An ActorBuilder instance
// is created by calling ActorSystem.BuildActor.
type ActorBuilder struct {
	actorBuilder
}

// Data required by concrete
// actor builder.
type actorBuilder struct {
	as    *ActorSystem
	actor *Actor
	err   error
}

// Build a basic actor with a message handling function. The
// actor invokes the message handling function when it reads
// a message from its mailbox.
func (as *ActorSystem) BuildActor(name string, doFunc func(*Actor, ActorMsg)) *ActorBuilder {
	a := Actor{
		as,
		make(chan ActorMsg, 10),
		doFunc,
		func(*Actor) {}, // enter
		func(*Actor) {}, // exit
		mainLoop,
		name,
		0,
		false,
	}
	err := a.validName()
	return &ActorBuilder{
		actorBuilder{
			as,
			&a,
			err,
		}}
}

// Add an enter function to the actor. The enter function
// gets called once when the actor starts.
func (b *ActorBuilder) WithEnter(enterFunc func(*Actor)) *ActorBuilder {
	if b.err == nil {
		b.actor.enterFunc = enterFunc
	}
	return b
}

// Add an exit function to the actor. The exit function
// gets called once when the actor exits.
func (b *ActorBuilder) WithExit(exitFunc func(*Actor)) *ActorBuilder {
	if b.err == nil {
		b.actor.exitFunc = exitFunc
	}
	return b
}

// Private method to exclude an actor from the
// actor system directory. Can be used to create
// transient actors.
func (b *ActorBuilder) withHidden() *ActorBuilder {
	if b.err == nil {
		b.actor.hidden = true
	}
	return b
}

// Turn an actor into a pool. When the actor starts,
// N copies of the actor are created. When a message
// is read from the actor's mailbox it is assigned
// to one of the copies on a round robin basis.
func (b *ActorBuilder) WithPool(num int) *ActorBuilder {
	if b.err != nil {
		return b
	}
	if num < 1 {
		b.err = fmt.Errorf("Pool must have at least 1 actor (%v)", num)
		return b
	}
	worker := b.actor
	name := worker.name
	pool := make([]Actor, num)
	idx := 0
	// Create a vanilla actor with a loop starter
	b.actor = &Actor{
		b.as,
		make(chan ActorMsg, 10),
		// 'do' function loops round actors
		func(ac *Actor, am ActorMsg) {
			pool[idx].mailbox <- am
			idx++
			if idx >= len(pool) {
				idx = 0
			}
		},
		nil, // enter
		func(*Actor) { // on exit, kill theworkers
			for _, a := range pool {
				a.Ref().SendMsg(newActorMsg(MsgTypePoison, "", nil))
			}
		},
		// main loops starts worker loops before starting its own
		func(a *Actor) {
			for i := range pool {
				// log.Printf("%v Starting worker loop %v", a.name, i)
				go pool[i].loopFunc(&pool[i])
			}
			// log.Printf("Starting pool loop")
			mainLoop(a)
		},
		name,
		0,
		false,
	}
	// create the workers with the decorated functions
	for idx := range pool {
		pool[idx] = Actor{
			b.as,
			make(chan ActorMsg, 10),
			worker.doFunc,
			worker.enterFunc,
			worker.exitFunc,
			mainLoop,
			fmt.Sprintf("%v#%v", name, idx),
			idx,
			true, // workers are hidden
		}
	}
	return b
}

// This must be the last call in the builder chain.
// It registers the actor in the actor system
// directory, calls the actor entry function, and
// starts the actor reading from its mailbox.
func (b *actorBuilder) Run() (*ActorRef, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.actor.run(b.as)
}
