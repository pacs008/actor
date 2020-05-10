// perf_test
package actor

import (
	"testing"
)

func BenchmarkLocalHello(b *testing.B) {
	for i := 0; i < b.N; i++ {
		localHello("Hello")
	}
}

func BenchmarkRemoteHello(b *testing.B) {
	as := NewActorSystem()
	server, _ := as.NewActor("greeter", func(ac ActorContext, msg ActorMsg) {
		call := msg.(CallRequest)
		call.CallResponse(call.Parameters().(string)+"X", nil)
	})
	for i := 0; i < b.N; i++ {
		server.Call("method", "hello", 10)
	}
}

func localHello(str string) string {
	return str + "X"
}
