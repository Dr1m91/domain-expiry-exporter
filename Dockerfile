FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o /out/domain-expiry-exporter ./cmd/domain-expiry-exporter

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/domain-expiry-exporter /usr/local/bin/domain-expiry-exporter
EXPOSE 9222
ENTRYPOINT ["/usr/local/bin/domain-expiry-exporter"]
