# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /go/src/app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY modules/model-deployment/. ./modules/model-deployment/
COPY internal/. ./internal/

WORKDIR /go/src/app/modules/model-deployment
RUN go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app
COPY --from=builder /go/src/app/modules/model-deployment/main ./modules/model-deployment/main

WORKDIR /app/modules/model-deployment
CMD ["./main"]
