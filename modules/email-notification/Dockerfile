# Build stage
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /go/src/app

# Copy the Go module files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy only the necessary directories and files
COPY modules/email-notification/. ./modules/email-notification/
COPY modules/utils/. ./modules/utils/

# Change working directory to build the application
WORKDIR /go/src/app/modules/email-notification

# Build the Go application
RUN go build -o main .

# Final stage
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
RUN mkdir -p /app/modules/email-notification
COPY --from=builder /go/src/app/modules/email-notification/main ./modules/email-notification/main

# Set the entry point for the container
WORKDIR /app/modules/email-notification
CMD ["./main"]
