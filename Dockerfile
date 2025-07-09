FROM ubuntu:24.04

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get install -y -qq \
    apt-transport-https \
    build-essential \
    ca-certificates \
    curl \
    git \
    gnupg-agent \
    jq \
    make \
    software-properties-common \
    sudo \
    zsh \
    > /dev/null 2>&1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*
ARG TARGETARCH
RUN curl --fail -s https://github.com/mikefarah/yq/releases/latest/download/yq_linux_${TARGETARCH}} -o /usr/local/bin/yq && chmod +x /usr/local/bin/yq

ENV GO_VERSION=1.24.4
RUN curl --fail -s https://dl.google.com/go/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz | tar -C /usr/local -xz

SHELL ["/usr/bin/zsh", "-o", "pipefail", "-c"]

CMD /usr/bin/zsh
