FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /sakura-agent ./cmd/agent

FROM alpine:3.19

RUN apk add --no-cache ca-certificates ipmitool lshw dmidecode

WORKDIR /app
COPY --from=builder /sakura-agent .

EXPOSE 8081

CMD ["./sakura-agent"]
