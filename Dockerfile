ARG IMAGE=ubuntu
ARG TAG=24.04
FROM ${IMAGE}:${TAG}

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
  apt-get install -y -qq \
  build-essential \
  curl \
  git \
  libsqlite3-dev \
  make \
  sqlite3 \
  sudo \
  unzip \
  wget \
  xz-utils \
  zsh \
  > /dev/null 2>&1 && \
  apt-get clean && rm -rf /var/lib/apt/lists/*

ARG GO_VERSION=1.24.4
ARG TARGETARCH

RUN curl \
  --fail \
  https://dl.google.com/go/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz | tar -xz -C /usr/local
RUN curl \
  --fail \
  -o /usr/local/bin/yq \
  https://github.com/mikefarah/yq/releases/latest/download/yq_linux_${TARGETARCH} \
  && chmod +x /usr/local/bin/yq

ENV PATH=/usr/local/go/bin:$PATH
ENV PATH=/root/go/bin:$PATH

SHELL ["/usr/bin/zsh", "-o", "pipefail", "-c"]

CMD /usr/bin/zsh
