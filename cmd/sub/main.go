package main

import (
	"context"
	"github.com/nats-io/nats.go"
	"nats-tracing/internal/config"
	"nats-tracing/internal/logger"
	"nats-tracing/internal/trace"
	"time"
)

const (
	shutdownTimeout = time.Second * 5
	packageName     = "sub"
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

	//tr := otel.Tracer(packageName)
	//ctx, mainSpan := tr.Start(ctx, "main")
	//defer mainSpan.End()

	log.Info("nats connecting", logger.String("server", cfg.NatsServer))
	nc, err := nats.Connect(cfg.NatsServer)
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
