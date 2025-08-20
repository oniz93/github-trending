# --- Builder Stage ---
FROM golang:latest AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd/embedding-autoscaler .

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -o /app/main .

# --- Final Stage ---
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

CMD ["./main"]
