FROM golang:1.14.4 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -a -tags netgo -ldflags '-w -extldflags "-static"' -o app .

FROM alpine
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/app .
CMD ["./app"]
