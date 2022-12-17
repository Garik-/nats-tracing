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
	shutdownTimeout     = time.Second * 5
	natsConnectedURLKey = "nats.connected.url"
	name                = "pub"
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

	opts := []nats.Option{nats.Name("NATS Sample Tracing Publisher")}
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

	//js.Publish("ORDERS.scratch", []byte("hello"))

	err = testPublish(ctx, js)
	if err != nil {
		log.Error("js.PublishMsg", logger.Error(err))
	}

}

func testPublish(ctx context.Context, js nats.JetStreamContext) error {
	ctx, span := otel.Tracer(name).Start(ctx, "publish message", trace2.WithSpanKind(trace2.SpanKindProducer))
	defer span.End()

	msg := spanToMsg(ctx, "ORDERS.test", []byte("hello trololo"))

	_, err := js.PublishMsg(msg)
	return errors.Wrap(err, "js.PublishMsg")
}

// copy from http.Header.Clone()
func header(h propagation.HeaderCarrier) nats.Header {
	if h == nil {
		return nil
	}

	// Find total number of values.
	nv := 0
	for _, vv := range h {
		nv += len(vv)
	}
	sv := make([]string, nv) // shared backing array for headers' values
	h2 := make(nats.Header, len(h))
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

func spanToMsg(ctx context.Context, subject string, data []byte) *nats.Msg {
	prop := otel.GetTextMapPropagator()
	headers := make(propagation.HeaderCarrier)
	prop.Inject(ctx, headers)

	return &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  header(headers),
	}
}
