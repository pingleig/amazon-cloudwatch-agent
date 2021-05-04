# Build the binary
ARG GO_IMAGE=golang:latest
ARG CERT_IMAGE=ubuntu:latest
FROM $GO_IMAGE as builder

RUN mkdir -p /go/src/github.com/aws/amazon-cloudwatch-agent/
WORKDIR /go/src/github.com/aws/amazon-cloudwatch-agent/

ARG GO111MODULE="on"
ENV GO111MODULE=${GO111MODULE}

COPY . /go/src/github.com/aws/amazon-cloudwatch-agent/
RUN make build-for-docker

# Install cert and binaries
FROM $CERT_IMAGE as cert

RUN mkdir -p /opt/aws/amazon-cloudwatch-agent/etc
RUN mkdir -p /opt/aws/amazon-cloudwatch-agent/var
RUN apt-get update &&  \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /go/src/github.com/aws/amazon-cloudwatch-agent/build/bin/linux_amd64/ /opt/aws/amazon-cloudwatch-agent/bin

FROM scratch

COPY --from=cert /tmp /tmp
COPY --from=cert /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=cert /opt/aws/amazon-cloudwatch-agent /opt/aws/amazon-cloudwatch-agent

ENV RUN_IN_CONTAINER="True"
ENTRYPOINT ["/opt/aws/amazon-cloudwatch-agent/bin/start-amazon-cloudwatch-agent"]
