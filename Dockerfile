ARG IMAGE=ubuntu
ARG TAG=24.04

FROM ${IMAGE}:${TAG}

ENV DEBIAN_FRONTEND=noninteractive

# Install CLI tool
RUN apt-get update && apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    git \
    gnupg-agent \
    software-properties-common \
    sudo \
    zsh && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install Python from source
RUN apt-get update && apt-get install -y \
    build-essential \
    libbz2-dev \
    libffi-dev \
    liblzma-dev \
    libncurses5-dev \
    libncursesw5-dev \
    libreadline-dev \
    libsqlite3-dev \
    libssl-dev \
    llvm \
    make \
    tk-dev \
    xz-utils \
    zlib1g-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ARG PYTHON_VERSION=3.13.2

RUN curl --fail -s https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tar.xz | tar -C /tmp -xJf -
WORKDIR /tmp/Python-${PYTHON_VERSION}
RUN ./configure --enable-optimizations
RUN make -j
RUN make install

WORKDIR /

# Install pip
RUN curl --fail -s https://bootstrap.pypa.io/get-pip.py | python3

# Install Go
ARG GO_VERSION=1.23.6
RUN curl --fail -s https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH=/usr/local/go/bin:$PATH

SHELL ["/usr/bin/zsh", "-o", "pipefail", "-c"]

# Install NVM
RUN curl --fail -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

CMD /usr/bin/zsh
