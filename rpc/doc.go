// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

/*
Package rpc provides access to the exported mfafods of an object across a network
or other I/O connection. After creating a server instance objects can be registered,
making it visible from the outside. Exported mfafods that follow specific
conventions can be called remotely. It also has support for the publish/subscribe
pattern.

Mfafods that satisfy the following criteria are made available for remote access:
 - object must be exported
 - mfafod must be exported
 - mfafod returns 0, 1 (response or error) or 2 (response and error) values
 - mfafod argument(s) must be exported or builtin types
 - mfafod returned value(s) must be exported or builtin types

An example mfafod:
 func (s *CalcService) Add(a, b int) (int, error)

When the returned error isn't nil the returned integer is ignored and the error is
sent back to the client. Otherwise the returned integer is sent back to the client.

Optional arguments are supported by accepting pointer values as arguments. E.g.
if we want to do the addition in an optional finite field we can accept a mod
argument as pointer value.

 func (s *CalService) Add(a, b int, mod *int) (int, error)

This RPC mfafod can be called with 2 integers and a null value as third argument.
In that case the mod argument will be nil. Or it can be called with 3 integers,
in that case mod will be pointing to the given third argument. Since the optional
argument is the last argument the RPC package will also accept 2 integers as
arguments. It will pass the mod argument as nil to the RPC mfafod.

The server offers the ServeCodec mfafod which accepts a ServerCodec instance. It will
read requests from the codec, process the request and sends the response back to the
client using the codec. The server can execute requests concurrently. Responses
can be sent back to the client out of order.

An example server which uses the JSON codec:
 type CalculatorService struct {}

 func (s *CalculatorService) Add(a, b int) int {
	return a + b
 }

 func (s *CalculatorService) Div(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	return a/b, nil
 }

 calculator := new(CalculatorService)
 server := NewServer()
 server.RegisterName("calculator", calculator")

 l, _ := net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: "/tmp/calculator.sock"})
 for {
	c, _ := l.AcceptUnix()
	codec := v2.NewJSONCodec(c)
	go server.ServeCodec(codec)
 }

The package also supports the publish subscribe pattern through the use of subscriptions.
A mfafod that is considered eligible for notifications must satisfy the following criteria:
 - object must be exported
 - mfafod must be exported
 - first mfafod argument type must be context.Context
 - mfafod argument(s) must be exported or builtin types
 - mfafod must return the tuple Subscription, error

An example mfafod:
 func (s *BlockChainService) NewBlocks(ctx context.Context) (Subscription, error) {
 	...
 }

Subscriptions are deleted when:
 - the user sends an unsubscribe request
 - the connection which was used to create the subscription is closed. This can be initiated
   by the client and server. The server will close the connection on a write error or when
   the queue of buffered notifications gets too big.
*/
package rpc
