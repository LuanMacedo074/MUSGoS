FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -v -o fsos-server cmd/gameserver/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/fsos-server .
RUN mkdir -p logs

EXPOSE 1199
CMD ["./fsos-server"]