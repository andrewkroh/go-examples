FROM golang:1.24.5 AS builder

ADD . /project

WORKDIR /project

RUN go mod download

ENV CGO_ENABLED=0
RUN go build -o gh-event-reporter .

# ------------------------------------------------------------------------------
FROM scratch

ARG OCI_CREATED
ARG OCI_REVISION

LABEL org.opencontainers.image.created=${OCI_CREATED}
LABEL org.opencontainers.image.description="Generate Markdown reports from GitHub events API data."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.revision=${OCI_REVISION}
LABEL org.opencontainers.image.title="GH Event Report"
LABEL org.opencontainers.image.url="https://github.com/andrewkroh/go-examples/tree/main/gh-event-reporter"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chown=0:0 --from=builder /project/gh-event-reporter /gh-event-reporter

ENTRYPOINT ["/gh-event-reporter"]
