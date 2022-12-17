package trace

import (
	"context"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"nats-tracing/internal/logger"
	"time"
)

const (
	natsConnectedURLKey = "nats.connected.url"
)

type ProviderConfig struct {
	JaegerAgentHost string
	JaegerAgentPort string

	ServiceID      string
	ServiceName    string
	ServiceVersion string
	EnvName        string
}

func newJaegerProvider(config *ProviderConfig) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exporter, err := jaeger.New(
		jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(config.JaegerAgentHost),
			jaeger.WithAgentPort(config.JaegerAgentPort),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "jaeger.New")
	}

	return tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exporter),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.DeploymentEnvironmentKey.String(config.EnvName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.ServiceInstanceIDKey.String(config.ServiceID),
		)),
	), nil
}

func Init(config *ProviderConfig) (func(ctx context.Context, shutdownTimeout time.Duration), error) {
	tp, err := newJaegerProvider(config)
	if err != nil {
		return nil, errors.Wrap(err, "newJaegerProvider")
	}

	otel.SetTracerProvider(tp)

	tc := propagation.TraceContext{}
	// Register the TraceContext propagator globally.
	otel.SetTextMapPropagator(tc)

	return func(ctx context.Context, shutdownTimeout time.Duration) {
		log := logger.Get()
		ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal("trace provider shutdown", logger.Error(err))
		}
	}, nil
}

func SetupConnOptions(ctx context.Context, tr oteltrace.Tracer, opts []nats.Option) []nats.Option {
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		_, span := tr.Start(ctx, "Disconnect Error Handler",
			oteltrace.WithAttributes(
				attribute.String(natsConnectedURLKey, nc.ConnectedUrl()),
			))

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		_, span := tr.Start(ctx, "Reconnect Handler",
			oteltrace.WithAttributes(
				attribute.String(natsConnectedURLKey, nc.ConnectedUrl()),
			))
		span.End()
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		_, span := tr.Start(ctx, "Closed Handler",
			oteltrace.WithAttributes(
				attribute.String(natsConnectedURLKey, nc.ConnectedUrl()),
			))
		span.End()

		logger.Get().Info("nats close handler")
	}))
	return opts
}
