# --- Builder Stage ---
# Use the official Go image as a builder.
FROM golang:latest AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies.
COPY go.mod ./
COPY go.sum ./
COPY internal ./internal
RUN go mod download

# Copy the rest of the application source code.
COPY cmd/crawler .

# Build the Go application.
# -o /app/main specifies the output file.
# CGO_ENABLED=0 is important for creating a static binary for Alpine.
# -ldflags "-s -w" strips debug symbols to make the binary smaller.
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -o /app/main .

# --- Final Stage ---
# Use a minimal Alpine image for the final container.
FROM alpine:latest

# Set the working directory.
WORKDIR /root/

# Copy the built binary from the builder stage.
COPY --from=builder /app/main .

# (Optional) Copy any config files if needed.
# COPY config.yml .

# Command to run the application.
CMD ["./main"]
