# Build stage
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /go/src/app

# Copy the Go module files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy only the necessary directories and files
COPY go-modules/sbc-configuration/. ./go-modules/sbc-configuration/
COPY go-modules/sbc-configuration/. ./go-modules/sbc-configuration/
COPY go-modules/utils/. ./go-modules/utils/

# Change working directory to build the application
WORKDIR /go/src/app/go-modules/sbc-configuration

# Build the Go application
RUN go build -o main .

# Final stage
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /go/src/app/go-modules/sbc-configuration/main .

# Set the entry point for the container
CMD ["./main"]
