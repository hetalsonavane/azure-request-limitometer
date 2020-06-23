FROM golang:1.14-alpine as builder

RUN apk update && apk add --no-cache git libc-dev gcc

COPY . /build

WORKDIR /build
RUN go build -o limitometer ./cmd/limitometer

FROM alpine:3.12.0

RUN apk update && \
    apk add --no-cache \
      bash \
      ca-certificates

WORKDIR /root

COPY --from=builder /build/limitometer /bin/limitometer

ENTRYPOINT ["/bin/limitometer"]
