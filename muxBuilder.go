// mux2Builder
package actor

import (
	"fmt"
	// log "github.com/sirupsen/logrus"
)

type mux2Builder struct {
	as  *ActorSystem
	mux *Mux
	err error
}

// initFunc takes the outbound channel as an argument and returns the inbound
func (as *ActorSystem) BuildMux(name string, initFunc func(chan<- []byte) chan []byte) *mux2Builder {
	builder := &mux2Builder{
		as,
		&Mux{
			Actor{
				as,
				make(chan ActorMsg, 10),
				nil, // no user-definable do, enter, exit fns
				nil,
				nil,
				nil, // don't use the Actor main loop
				name,
				0,
				false,
			},
			make(map[string]CacheEntry), // cache
			initFunc,
			nil, // kb key builder
			func(_ interface{}) bool { return false }, // reqPassThru
			func(_ interface{}) bool { return false }, // rspPassThru
			ActorRef{},            // null passthruRspActor
			make(chan []byte, 10), // reqChan
			nil,                   // rspChan
			nil,                   // tick
			5000,                  // timeout
		},
		nil,
	}
	// set up the actor's enter function to initialize the resource
	// builder.mux.enterFunc = func(_ ActorContext) {
	// 	log.Debugf("%v Into exterFunc ", builder.mux.Name())
	// 	builder.mux.rspChan = initFunc(builder.mux.reqChan)
	// 	log.Debugf("%v rspChan %v ", builder.mux.Name(), builder.mux.rspChan)
	// }

	return builder

}

// add key builder
func (b *mux2Builder) WithKeyBuilder(builderFunc func(interface{}) string) *mux2Builder {
	if b.err == nil {
		b.mux.kb = builderFunc
	}
	return b
}

// add timeout
func (b *mux2Builder) WithTimeout(timeout int) *mux2Builder {
	if b.err == nil {
		if timeout <= 0 {
			b.err = fmt.Errorf("Invalid timeout %v: must be > 0", timeout)
		} else {
			b.mux.timeout = timeout
		}
	}
	return b
}

// add pass-thru function (request)
func (b *mux2Builder) WithReqPassthru(passthru func(interface{}) bool) *mux2Builder {
	if b.err == nil {
		b.mux.reqPassthru = passthru
	}
	return b
}

// add pass-thru function (response)
func (b *mux2Builder) WithRspPassthru(passthru func(interface{}) bool, passthruRspActor ActorRef) *mux2Builder {
	if b.err == nil {
		b.mux.reqPassthru = passthru
		b.mux.passthruRspActor = passthruRspActor
	}
	return b
}

// last call in chain - start it up
func (b *mux2Builder) Run() (*ActorRef, error) {
	if b.mux.kb == nil {
		b.err = fmt.Errorf("Mux must have KeyBuilder defined")
	}
	if b.err != nil {
		return nil, b.err
	}
	return b.mux.run(b.as)
}
