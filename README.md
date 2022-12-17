# Tracing services through a message queue. OpenTelemetry, NATS.

## OpenTelemetry
https://opentelemetry.io/docs/concepts/signals/traces/

> ### Trace Context
> Trace Context is metadata about trace spans that provides correlation between spans across service and process boundaries. For example, let’s say that Service A calls Service B and you want to track the call in a trace. In that case, OpenTelemetry will use Trace Context to capture the ID of the trace and current span from Service A, so that spans created in Service B can connect and add to the trace.
>
> This is known as Context Propagation.
>
> ### Context Propagation
> Context Propagation is the core concept that enables Distributed Tracing. With Context Propagation, Spans can be correlated with each other and assembled into a trace, regardless of where Spans are generated. We define Context Propagation by two sub-concepts: Context and Propagation.
> 
> A **Context** is an object that contains the information for the sending and receiving service to correlate one span with another and associate it with the trace overall.
> 
> **Propagation** is the mechanism that moves Context between services and processes. By doing so, it assembles a Distributed Trace. It serializes or deserializes Span Context and provides the relevant Trace information to be propagated from one service to another. We now have what we call: **Trace Context**.

OK let's create a propagation instance and initialize it
```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

tc := propagation.TraceContext{}
// Register the TraceContext propagator globally.
otel.SetTextMapPropagator(tc)
```

In the service A whose context we want to transmit
```go
// GetTextMapPropagator returns the global TextMapPropagator.
prop := otel.GetTextMapPropagator()
// HeaderCarrier adapts http.Header to satisfy the TextMapCarrier interface.
headers := make(propagation.HeaderCarrier)
prop.Inject(ctx, headers)
```
after that we have to somehow pass these headers, in the request body, in the request headers, it all depends on your implementation.

In service B in which we want to get the context
```go
var headers propagation.HeaderCarrier
// we get the headers and convert them to HeaderCarrier...
prop := otel.GetTextMapPropagator()
// Extract reads cross-cutting concerns from the carrier into a Context.
ctx = prop.Extract(ctx, headers)
```

## NATS
```go
// Simple Async Subscriber
nc.Subscribe("foo", func(m *nats.Msg) {
    fmt.Printf("Received a message: %s\n", string(m.Data))
})

// Header represents the optional Header for a NATS message,
// based on the implementation of http.Header.
type Header map[string][]string

// Msg represents a message delivered by NATS. This structure is used
// by Subscribers and PublishMsg().
type Msg struct {
	Header  Header
}
```
it is not difficult to notice that propagation.HeaderCarrier and nats.Header are based on the http.Header implementation.
Therefore, to copy data from one structure to another, I took the implementation of the [http.Header.Clone()](https://cs.opensource.google/go/go/+/refs/tags/go1.19.4:src/net/http/header.go;l=94) method

## Example
```BASH
docker-compose build
make env-up
make run
```

http://localhost:16686/
