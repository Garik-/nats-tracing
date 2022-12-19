package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucsky/cuid"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	"nats-tracing/internal/logger"
)

const (
	subject     = "ORDERS.test"
	repeatDelay = time.Second / 2
)

type service struct {
	js nats.JetStreamContext
}

func runDone(ctx context.Context, cancel context.CancelFunc) error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-done:
			cancel() // close parent context
			return nil
		}
	}
}

func newService(js nats.JetStreamContext) *service {
	return &service{
		js: js,
	}
}

func (s *service) run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errGroup, errCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		return runDone(errCtx, cancel)
	})

	errGroup.Go(func() error {
		log := logger.Get()
		for {
			select {
			case <-errCtx.Done():
				return nil
			default:
			}

			err := s.publishEvent(errCtx)
			if err != nil {
				log.Error("publishEvent", logger.Error(err))
			}

			delay(ctx, repeatDelay)
		}
	})

	return errGroup.Wait()
}

func (s *service) publishEvent(ctx context.Context) error {
	id := cuid.New()
	ctx, span := otel.Tracer(name).Start(ctx, id, trace.WithSpanKind(trace.SpanKindProducer))

	defer span.End()

	msg := newMsg(ctx, subject, []byte(id))

	_, err := s.js.PublishMsg(msg)
	if err != nil {
		return errors.Wrap(err, "js.PublishMsg")
	}

	logger.Get().Debug("publishEvent", logger.String("id", id))

	return nil
}

func delay(ctx context.Context, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
