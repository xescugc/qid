FROM golang:1.25.1 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /pikoci .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates git jq curl openssl docker-cli
COPY --from=builder /pikoci /usr/local/bin/pikoci
ENTRYPOINT ["pikoci"]
