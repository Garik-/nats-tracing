package main

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// copy from http.Header.Clone().
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

func newMsg(ctx context.Context, subject string, data []byte) *nats.Msg {
	prop := otel.GetTextMapPropagator()
	headers := make(propagation.HeaderCarrier)
	prop.Inject(ctx, headers)

	return &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  header(headers),
	}
}
