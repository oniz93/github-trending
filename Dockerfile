# syntax=docker/dockerfile:1.4
FROM golang:1.21-alpine AS builder

# Set Go environment variables
ENV GO111MODULE=on

COPY go.mod go.sum .
RUN go mod download
COPY . .

FROM builder AS api-builder
WORKDIR /
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -ldflags "-s -w" -o /app/api-server github-trending/cmd/api

FROM builder AS crawler-builder
WORKDIR /
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -ldflags "-s -w" -o /app/crawler-service github-trending/cmd/crawler

FROM builder AS discovery-builder
WORKDIR /
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -ldflags "-s -w" -o /app/discovery-service github-trending/cmd/discovery

FROM builder AS processor-builder
WORKDIR /
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -ldflags "-s -w" -o /app/processor-service github-trending/cmd/processor

FROM builder AS scheduler-builder
WORKDIR /
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -ldflags "-s -w" -o /app/scheduler-service github-trending/cmd/scheduler

# Final stages for each service
FROM alpine:latest AS api
WORKDIR /root/
COPY --from=builder /app/api-server .
CMD ["./api-server"]

FROM alpine:latest AS crawler
WORKDIR /root/
COPY --from=builder /app/crawler-service .
CMD ["./crawler-service"]

FROM alpine:latest AS discovery
WORKDIR /root/
COPY --from=builder /app/discovery-service .
CMD ["./discovery-service"]

FROM alpine:latest AS processor
WORKDIR /root/
COPY --from=builder /app/processor-service .
CMD ["./processor-service"]

FROM alpine:latest AS scheduler
WORKDIR /root/
COPY --from=builder /app/scheduler-service .
CMD ["./scheduler-service"]