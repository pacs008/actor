package actor

// import (
// 	"fmt"
// )

// Choose a destination ActorRef based on
// the message contents.
type RouteFunc func(interface{}) *ActorRef

type routeBuilder struct {
	as     *ActorSystem
	name   string
	routes []RouteFunc
}

// Convenience method to quickly create a router without using builder.
// A router is simply an actor that forwards a message to another actor
// depending on the contents of the message.
func (as *ActorSystem) NewRouter(name string, fn RouteFunc) (*ActorRef, error) {
	return as.BuildRouter(name).AddRoute(fn).Run()
}

// Build a router.
// A router is simply an actor that forwards a message to another actor
// depending on the contents of the message.
func (as *ActorSystem) BuildRouter(name string) *routeBuilder {
	return &routeBuilder{
		as,
		name,
		[]RouteFunc{},
	}
}

// Add a route to the router.
func (rb *routeBuilder) AddRoute(fn RouteFunc) *routeBuilder {
	rb.routes = append(rb.routes, fn)
	return rb
}

// Run the router.
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
