# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod ./
RUN go mod tidy && go mod download

COPY . .

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=off
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o alertbot cmd/server/main.go

# Final stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/alertbot .
COPY --from=builder /app/configs ./configs

EXPOSE 8080

CMD ["./alertbot"]