FROM golang:1.22.4-alpine3.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /workplace ./cmd/server/main.go

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /root/

COPY --from=builder /workplace .
COPY --from=builder /app/.env* .

EXPOSE 8080

CMD ["./workplace"]
