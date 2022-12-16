FROM alpine:3.17 as builder
ENV NATSCLI_VERSION 0.0.35
RUN apk --no-cache add bash curl libarchive-tools
RUN curl -L nats.zip https://github.com/nats-io/natscli/releases/download/v${NATSCLI_VERSION}/nats-${NATSCLI_VERSION}-linux-amd64.zip | \
    bsdtar -xvf- && mv nats-${NATSCLI_VERSION}-linux-amd64/nats /nats

FROM alpine:3.17
RUN apk --no-cache add bash
COPY --from=builder /nats /usr/local/bin/nats
RUN chmod +x /usr/local/bin/nats
