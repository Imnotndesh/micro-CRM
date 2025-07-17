FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o microCRM "./cmd/micro-crm/main.go"

FROM alpine:latest
WORKDIR /app
RUN apk add bash
COPY --from=builder /app/microCRM .
EXPOSE 9080
CMD ["./microCRM"]