// actor project doc.go

/*
The actor package provides a lightweight Actor framework for go.
It is consciously based on the Akka framework, which in turn derives from Erlang.
The main restriction of actor is that it is not a distributed framework -
there are plenty of distributed frameworks in which the actor package can work.

The actor framework provides a simple model to implement asynchronous concurrent
processing. Actors deal with one message at a time, so developers do not need to
handle locking and synchronization. Parallelism is achieved by running several
actors in parallel. The actor framework provides a pooling mechanism to simplify
this.

Actors encapsulate functionality and state. They communicate via
message passing. Developers define functions to handle the messages that an
actor receives. An actor can receive multiple message types.

As well as point-to-point message passing, the actor framework also supports
a pub-sub model. Actors subscribing to a topic receive messages in the same
handler function as normal messages.

Although the framework focuses on asynchronous communications, it also supports
a synchronous call mechanism. Normal actors can also act as RPC servers - a call
is received like a normal message with a response method.

The package provides a simple directory service to make actors discoverable.

*/
package actor
