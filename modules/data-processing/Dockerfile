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
COPY modules/data-processing/. ./modules/data-processing/
COPY modules/data-processing/. ./modules/data-processing/
COPY modules/utils/. ./modules/utils/

# Change working directory to build the application
WORKDIR /go/src/app/modules/data-processing

# Build the Go application
RUN go build -o main .

# Final stage
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
RUN mkdir -p /app/modules/data-processing
COPY --from=builder /go/src/app/modules/data-processing/main ./modules/data-processing/main
COPY --from=builder /go/src/app/certs ./certs

# Set the entry point for the container
WORKDIR /app/modules/data-processing
CMD ["./main"]
