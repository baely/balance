FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./go.mod
COPY go.sum ./go.sum

RUN go mod download

COPY . .

RUN go build -o /balance

FROM alpine

COPY --from=builder /balance /balance

ENTRYPOINT ["/balance"]
