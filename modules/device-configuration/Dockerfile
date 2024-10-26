# Build stage
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /go/src/app

# Copy the Go module files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy only the necessary directories and files
COPY certs/ca-crt.pem ./certs/ca-crt.pem
COPY certs/client-crt.pem ./certs/client-crt.pem
COPY certs/client-key.pem ./certs/client-key.pem
COPY modules/device-configuration/. ./modules/device-configuration/
COPY modules/device-configuration/. ./modules/device-configuration/
COPY modules/utils/. ./modules/utils/

# Change working directory to build the application
WORKDIR /go/src/app/modules/device-configuration

# Build the Go application
RUN go build -o main .

# Final stage
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /go/src/app/modules/device-configuration/main ./modules/device-configuration/main
COPY --from=builder /go/src/app/certs ./certs

# Set the entry point for the container
WORKDIR /app/modules/device-configuration
CMD ["./main"]
