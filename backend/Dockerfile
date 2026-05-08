FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/bin/api ./cmd/api

RUN go build -o /app/bin/worker ./cmd/worker

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/worker /app/worker

COPY --from=builder /app/config/iam.yaml /app/config/iam.yaml
COPY --from=builder /app/templates /app/templates

ENV TEMPORAL_HOST=host.docker.internal:7233
ENV DB_PATH=/app/data/workflow_engine.db
ENV LISTEN_ADDR=:8080
ENV IAM_PATH=/app/config/iam.yaml

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/api"]
