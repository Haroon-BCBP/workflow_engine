FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o api-server ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build -o worker-server ./cmd/worker

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/api-server .
COPY --from=builder /app/worker-server .
COPY --from=builder /app/config/iam.yaml ./config/iam.yaml
COPY --from=builder /app/templates ./templates

ENV TEMPORAL_HOST=temporal:7233
ENV DB_PATH=/app/data/workflow_engine.db
ENV LISTEN_ADDR=:8080
ENV IAM_PATH=config/iam.yaml

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./api-server"]
