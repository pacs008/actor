package actor

// import (
// 	"fmt"
// )

type routeFunc func(interface{}) *ActorRef

type routeBuilder struct {
	as     *ActorSystem
	name   string
	routes []routeFunc
}

// convenience method to quickly create a router without using builder
func (as *ActorSystem) NewRouter(name string, fn routeFunc) (*ActorRef, error) {
	return as.BuildRouter(name).AddRoute(fn).Run()
}

func (as *ActorSystem) BuildRouter(name string) *routeBuilder {
	return &routeBuilder{
		as,
		name,
		[]routeFunc{},
	}
}

func (rb *routeBuilder) AddRoute(fn routeFunc) *routeBuilder {
	rb.routes = append(rb.routes, fn)
	return rb
}

func (rb *routeBuilder) Run() (*ActorRef, error) {
	a := Actor{
		rb.as,
		make(chan ActorMsg, 10),
		func(ac *Actor, am ActorMsg) { // doFunc
			for _, route := range rb.routes {
				ref := route(am.Data())
				if ref != nil {
					ref.Forward(am)
					return
				}
			}
			rb.as.ToDeadLetter(am)
		},
		func(*Actor) {}, // enterFn
		func(*Actor) {}, // exitFn
		mainLoop,
		rb.name,
		0,
		false,
	}

	return a.run(rb.as)
}
