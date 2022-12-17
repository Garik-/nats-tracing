package main

import (
	"context"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	trace2 "go.opentelemetry.io/otel/trace"
	"nats-tracing/internal/config"
	"nats-tracing/internal/logger"
	"nats-tracing/internal/trace"
	"time"
)

const (
	shutdownTimeout = time.Second * 5
	name            = "sub"
)

func main() {
	disableLogger := logger.Enable()
	defer disableLogger()

	log := logger.Get()

	cfg, err := config.New("")
	if err != nil {
		log.Fatal("config create", logger.Error(err))
	}

	cfg.Print()
	err = cfg.Validate()
	if err != nil {
		log.Fatal("config validate", logger.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cancelTrace, err := trace.Init(&trace.ProviderConfig{
		JaegerAgentHost: cfg.JaegerAgentHost,
		JaegerAgentPort: cfg.JaegerAgentPort,
		ServiceID:       cfg.ServiceID,
		ServiceName:     cfg.ServiceName,
		ServiceVersion:  cfg.ServiceVersion,
	})

	if err != nil {
		log.Fatal("trace.Init", logger.Error(err))
	}

	defer cancelTrace(ctx, shutdownTimeout)

	opts := []nats.Option{nats.Name("NATS Sample Tracing Subscriber")}
	tr := otel.Tracer(name)
	ctx, mainSpan := tr.Start(ctx, "main")
	defer mainSpan.End()

	opts = append(opts, nats.ConnectHandler(func(nc *nats.Conn) {
		_, span := tr.Start(ctx, "ConnectHandler")
		span.End()
	}))

	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		_, span := tr.Start(ctx, "ClosedHandler")
		span.End()
	}))

	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err2 error) {
		if err != nil {
			_, span := tr.Start(ctx, "DisconnectErrHandler")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
		}
	}))

	log.Info("nats connecting", logger.String("server", cfg.NatsServer))
	nc, err := nats.Connect(cfg.NatsServer, opts...)
	if err != nil {
		log.Fatal("nats.Connect", logger.Error(err))
	}
	defer nc.Close()

	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))

	if err != nil {
		log.Fatal("nc.JetStream", logger.Error(err))
	}

	err = Subscribe(ctx, js, "ORDERS.*", "test-id",
		handlerFunc,
		nats.MaxAckPending(256),
		nats.DeliverLastPerSubject(),
		nats.AckAll(),
	)

	if err != nil {
		log.Fatal("Subscribe", logger.Error(err))
	}
}

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

// copy from http.Header.Clone()
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

	_, span := otel.Tracer(name).Start(ctx, "subject recive", trace2.WithSpanKind(trace2.SpanKindConsumer))
	defer span.End()

	return nil
}
