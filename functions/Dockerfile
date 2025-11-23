FROM rust:1.75-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache musl-dev
RUN cargo build --release --workspace

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/target/release/edge-runner /usr/local/bin/
EXPOSE 8002
CMD ["edge-runner"]
