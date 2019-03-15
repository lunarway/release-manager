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
 
FROM alpine:3.9
RUN   apk update && \
      apk add --no-cache \
      openssh-keygen bash openssh-client git ca-certificates
RUN ssh-keyscan github.com bitbucket.org >> /etc/ssh/ssh_known_hosts
WORKDIR /app

COPY ./ssh_config /etc/ssh/ssh_config
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/server .

ENTRYPOINT [ "./server", "start" ]

