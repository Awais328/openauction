FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY core/ ./core/
COPY enclaveapi/ ./enclaveapi/

COPY enclave/*.go ./enclave/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -ldflags '-extldflags "-static"' \
    -tags netgo \
    -o tee-auction-enclave \
    ./enclave

FROM amazonlinux:2023

ARG ENCLAVE_MAX_WORKERS=50
ENV ENCLAVE_MAX_WORKERS=${ENCLAVE_MAX_WORKERS}

RUN dnf update -y && \
    dnf install -y ca-certificates && \
    dnf clean all && \
    rm -rf /var/cache/dnf

COPY --from=builder /app/tee-auction-enclave /usr/local/bin/tee-auction-enclave

RUN chmod +x /usr/local/bin/tee-auction-enclave

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD echo '{"type":"ping","message":"health_check"}' | nc 127.0.0.1 5000 || exit 1

ENTRYPOINT ["/usr/local/bin/tee-auction-enclave"]

CMD []
