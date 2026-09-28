FROM golang:1.27.1-alpine AS builder

WORKDIR /app

COPY go.mod ./

COPY . .

RUN go build -o cloud-monitor .

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/cloud-monitor /app/cloud-monitor

CMD [ "/app/cloud-monitor" ]

