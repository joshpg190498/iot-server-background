# Build stage
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /go/src/app

# Copy the Go module files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy only the necessary directories and files
COPY modules/data-reception/. ./modules/data-reception/
COPY modules/utils/. ./modules/utils/

# Change working directory to build the application
WORKDIR /go/src/app/modules/data-reception

# Build the Go application
RUN go build -o main .

# Final stage
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the builder stage
RUN mkdir -p /app/modules/data-reception
COPY --from=builder /go/src/app/modules/data-reception/main ./modules/data-reception/main
COPY --from=builder /go/src/app/modules/data-reception/certs/. ./modules/data-reception/certs/

# Set the entry point for the container
WORKDIR /app/modules/data-reception
CMD ["./main"]
