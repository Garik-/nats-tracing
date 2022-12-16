#!/usr/bin/env bash
set -euo pipefail

if [[ -z "$NATS_HOST" ]]; then
    echo "Must provide NATS_HOST in environment" 1>&2
    exit 1
fi

if [[ -z "$NATS_PORT" ]]; then
    echo "Must provide NATS_PORT in environment" 1>&2
    exit 1
fi

sleep 10

NATS_STREAM_NAME=ORDERS

if [[ $(nats stream ls --server=${NATS_HOST}:${NATS_PORT} | grep ${NATS_STREAM_NAME}) ]];
    then
        echo "Nats:" "Stream ${NATS_STREAM_NAME} exist"
    else
        nats stream add ${NATS_STREAM_NAME} --subjects "ORDERS.*" \
                 --ack --max-msgs=-1 \
                 --max-bytes=-1 \
                 --max-age=1h \
                 --storage file \
                 --retention limits \
                 --max-msg-size=-1 \
                 --discard=old \
                 --max-msgs-per-subject=-1 \
                 --dupe-window="2m" \
                 --no-allow-rollup \
                 --no-deny-delete \
                 --no-deny-purge \
                 --replicas=1 \
                 --server=${NATS_HOST}:${NATS_PORT}
  fi
