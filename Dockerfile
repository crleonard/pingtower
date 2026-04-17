FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /pingtower ./cmd/server

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /pingtower /pingtower

EXPOSE 8080

ENV PINGTOWER_ADDR=:8080
ENV PINGTOWER_DATA_FILE=/data/pingtower.json

VOLUME ["/data"]

ENTRYPOINT ["/pingtower"]
