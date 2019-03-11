FROM golang:1.12.0 as builder
WORKDIR /app
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY generated/ ./generated/
COPY internal/ ./internal/
COPY cmd/server/ ./cmd/server/

RUN go build -o server cmd/server/main.go
 
FROM scratch
WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs
COPY --from=builder /app/server .

ENTRYPOINT [ "./server" ]

