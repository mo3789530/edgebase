FROM rust:1.75-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache musl-dev
RUN cargo build --release --workspace

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/target/release/sync-service /usr/local/bin/
COPY --from=builder /app/target/release/edge-agent /usr/local/bin/
EXPOSE 8001
CMD ["sync-service"]
