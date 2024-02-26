FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./go.mod
COPY go.sum ./go.sum

RUN go mod download

COPY . .

RUN go build -o /go/bin/app

FROM alpine

COPY --from=builder /go/bin/app /go/bin/app

ENTRYPOINT ["/go/bin/app"]
