FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache git
RUN go build -o bin/control-plane ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/control-plane /usr/local/bin/
EXPOSE 8000
CMD ["control-plane"]
