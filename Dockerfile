ARG IMAGE=ubuntu
ARG TAG=24.04

FROM ${IMAGE}:${TAG} AS python-builder

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get install -y -qq \
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
        zlib1g-dev \
        curl \
        > /dev/null 2>&1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

ENV PYTHON_VERSION=3.13.2
RUN curl --fail -s https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tar.xz | tar -C /tmp -xJf -
WORKDIR /tmp/Python-${PYTHON_VERSION}
RUN ./configure \
    --enable-optimizations \
    --without-system-libmpdec > /dev/null
RUN make -j > /dev/null 2>&1
RUN make install > /dev/null

FROM ${IMAGE}:${TAG} AS go-builder

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get install -y -qq \
        curl \
        > /dev/null 2>&1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

ENV GO_VERSION=1.23.6
RUN curl --fail -s https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar -C /usr/local -xz

FROM ${IMAGE}:${TAG} AS install

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get install -y -qq \
        apt-transport-https \
        build-essential \
        ca-certificates \
        curl \
        git \
        gnupg-agent \
        llvm \
        make \
        software-properties-common \
        sudo \
        zsh \
        > /dev/null 2>&1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

COPY --from=python-builder /usr/local/bin/python3 /usr/local/bin/python3
COPY --from=python-builder /usr/local/lib/python3.13 /usr/local/lib/python3.13
COPY --from=python-builder /usr/local/include/python3.13 /usr/local/include/python3.13
COPY --from=python-builder /usr/local/bin/pip3 /usr/local/bin/pip3

COPY --from=go-builder /usr/local/go /usr/local/go

ENV PATH=/usr/local/go/bin:$PATH
ENV PIP_ROOT_USER_ACTION=ignore

SHELL ["/usr/bin/zsh", "-o", "pipefail", "-c"]

# Install NVM
RUN curl --fail -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
COPY .nvmrc /root/.nvmrc
RUN touch /root/.bashrc && echo ". /root/.nvm/nvm.sh" >> /root/.bashrc
RUN touch /root/.zshrc && echo ". /root/.nvm/nvm.sh" >> /root/.zshrc

FROM ${IMAGE}:${TAG} AS final

COPY --from=install / /

CMD /usr/bin/zsh
