FROM --platform=$BUILDPLATFORM golang:1.18.0-stretch AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION

WORKDIR /build
RUN mkdir /build/tmp
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags="-s -w -X 'github.com/richardjennings/simple-ops/cmd.Version=${VERSION}'" -o simple-ops main.go

FROM scratch
COPY --chown=65534:0 --from=builder /build/simple-ops /
COPY --chown=65534:0 --from=builder /build/tmp /tmp
COPY --chown=65534:0 --from=builder /etc/ssl/certs/ /etc/ssl/certs/

WORKDIR /workdir
ENTRYPOINT ["/simple-ops"]
