// builder
package actor

import (
	"fmt"
	// "log"
)

type ActorBuilder interface {
	Run() (*ActorRef, error)
	WithEnter(func(ActorContext)) ActorBuilder
	WithExit(func(ActorContext)) ActorBuilder
	WithPool(int) ActorBuilder
}
type actorBuilder struct {
	as    *ActorSystem
	actor *Actor
	err   error
}

func (as *ActorSystem) BuildActor(name string, doFunc func(ActorContext, ActorMsg)) ActorBuilder {
	a := Actor{
		as,
		make(chan ActorMsg, 10),
		doFunc,
		func(ActorContext) {}, // enter
		func(ActorContext) {}, // exit
		mainLoop,
		name,
		0,
		false,
	}
	err := a.validName()
	return &actorBuilder{
		as,
		&a,
		err,
	}
}

// add enter function to actor
func (b actorBuilder) WithEnter(enterFunc func(ActorContext)) ActorBuilder {
	if b.err == nil {
		b.actor.enterFunc = enterFunc
	}
	return b
}

// add exit function to actor
func (b actorBuilder) WithExit(exitFunc func(ActorContext)) ActorBuilder {
	if b.err == nil {
		b.actor.exitFunc = exitFunc
	}
	return b
}

// hidden actor (private method)
func (b actorBuilder) withHidden() ActorBuilder {
	if b.err == nil {
		b.actor.hidden = true
	}
	return b
}

// turn a single actor into a pool
func (b actorBuilder) WithPool(num int) ActorBuilder {
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
		func(ac ActorContext, am ActorMsg) {
			pool[idx].mailbox <- am
			idx++
			if idx >= len(pool) {
				idx = 0
			}
		},
		nil, // enter
		func(ActorContext) { // on exit, kill theworkers
			for _, a := range pool {
				a.Self().SendMsg(newActorMsg(MsgTypePoison, "", nil))
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

// last call in chain - start it up
func (b actorBuilder) Run() (*ActorRef, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.actor.run(b.as)
}
