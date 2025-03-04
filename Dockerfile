ARG IMAGE=ubuntu
ARG TAG=24.04
ARG MAKE_JOBS=2

FROM ${IMAGE}:${TAG} AS node-builder
RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get install -y -qq \
        curl \
        unzip \
        > /dev/null 2>&1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL https://bun.sh/install | bash


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
    --without-system-libmpdec \
    --enable-loadable-sqlite-extensions \
    --with-ensurepip=install \
    --with-computed-gotos \
    > /dev/null

RUN if [ -z "$MAKE_JOBS" ]; then \
        if [ -f /proc/cpuinfo ]; then \
            CORES=$(grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 2); \
        elif command -v nproc > /dev/null; then \
            CORES=$(nproc 2>/dev/null || echo 2); \
        elif command -v sysctl > /dev/null; then \
            CORES=$(sysctl -n hw.ncpu 2>/dev/null || echo 2); \
        else \
            CORES=2; \
        fi && \
        MAKE_JOBS=$(( CORES < 1 ? 1 : CORES )); \
    fi && \
    echo "Building with ${MAKE_JOBS} jobs" && \
    make -j${MAKE_JOBS} \
        > /dev/null 2>&1 && \
    make install > /dev/null

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
        jq \
        make \
        software-properties-common \
        sudo \
        zsh \
        > /dev/null 2>&1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

COPY --link --from=python-builder /usr/local/bin/python3 /usr/local/bin/python
COPY --link --from=python-builder /usr/local/bin/python3 /usr/local/bin/python3
COPY --link --from=python-builder /usr/local/lib/python3.13 /usr/local/lib/python3.13
COPY --link --from=python-builder /usr/local/include/python3.13 /usr/local/include/python3.13
COPY --link --from=python-builder /usr/local/bin/pip3 /usr/local/bin/pip3
ARG TARGETARCH
RUN curl --fail -s -o /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_${TARGETARCH} /usr/local/bin/yq

COPY --from=go-builder /usr/local/go /usr/local/go

COPY --link --from=node-builder /root/.bun/bin/bun /usr/local/bin/bun
COPY --link --from=node-builder /root/.bun/bin/bun /usr/local/bin/node
COPY --link --from=node-builder /root/.bun/bin/bun /usr/local/bin/npm
COPY --link --from=node-builder /root/.bun/bin/bunx /usr/local/bin/npx

ENV PATH=/usr/local/go/bin:$PATH
ENV PIP_ROOT_USER_ACTION=ignore

SHELL ["/usr/bin/zsh", "-o", "pipefail", "-c"]

CMD /usr/bin/zsh
