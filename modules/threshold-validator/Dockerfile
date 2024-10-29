# Build stage
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /go/src/app

# Copy the Go module files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy only the necessary directories and files
COPY modules/threshold-validator/. ./modules/threshold-validator/
COPY modules/utils/. ./modules/utils/

# Change working directory to build the application
WORKDIR /go/src/app/modules/threshold-validator

# Build the Go application
RUN go build -o main .

# Final stage
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
RUN mkdir -p /app/modules/threshold-validator
COPY --from=builder /go/src/app/modules/threshold-validator/main ./modules/threshold-validator/main

# Set the entry point for the container
WORKDIR /app/modules/threshold-validator
CMD ["./main"]
