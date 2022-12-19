package main

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"nats-tracing/internal/logger"
)

type SubscribeHandler func(ctx context.Context, msg *nats.Msg) error

func Subscribe(
	ctx context.Context,
	stream nats.JetStream,
	subject, consumerID string,
	handler SubscribeHandler,
	opts ...nats.SubOpt,
) (err error) {
	sub, err := stream.QueueSubscribeSync(subject, consumerID, opts...)
	if err != nil {
		return errors.Wrap(err, "stream.QueueSubscribeSync")
	}

	err = handleSubscription(ctx, sub, handler)

	return err
}

func handleSubscription(ctx context.Context, sub *nats.Subscription, handler SubscribeHandler) error {
	for {
		select {
		case <-ctx.Done():
			return sub.Unsubscribe()
		default:
		}

		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return errors.Wrap(err, "sub.NextMsgWithContext")
		}

		err = handler(ctx, msg)
		if err != nil {
			return errors.Wrap(err, "cannot handle message")
		}
	}
}

// copy from http.Header.Clone().
func header(h nats.Header) propagation.HeaderCarrier {
	if h == nil {
		return nil
	}

	// Find total number of values.
	nv := 0
	for _, vv := range h {
		nv += len(vv)
	}

	sv := make([]string, nv) // shared backing array for headers' values
	h2 := make(propagation.HeaderCarrier, len(h))

	for k, vv := range h {
		if vv == nil {
			// Preserve nil values. ReverseProxy distinguishes
			// between nil and zero-length header values.
			h2[k] = nil
			continue
		}

		n := copy(sv, vv)
		h2[k] = sv[:n:n]
		sv = sv[n:]
	}

	return h2
}

func handlerFunc(ctx context.Context, msg *nats.Msg) error {
	log := logger.Get()

	defer func() {
		if errAck := msg.Ack(); errAck != nil {
			log.Error("msg.Ack", logger.Error(errAck))
		}
	}()

	log.Debug("recv", logger.String("data", string(msg.Data)))

	prop := otel.GetTextMapPropagator()
	headers := header(msg.Header)
	ctx = prop.Extract(ctx, headers)

	_, span := otel.Tracer(packageName).Start(ctx, "subject receive", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	return nil
}
