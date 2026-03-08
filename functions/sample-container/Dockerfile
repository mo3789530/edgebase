FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go build -o /out/sample-container .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates

COPY --from=builder /out/sample-container /usr/local/bin/sample-container

ENV PORT=8080
EXPOSE 8080

CMD ["sample-container"]
