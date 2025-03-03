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
    --without-system-libmpdec \
    --enable-loadable-sqlite-extensions \
    --enable-experimental-jit=yes \
    --with-ensurepip=install \
    --with-lto=full \
    --with-computed-gotos
    # > /dev/null

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
    make -j${MAKE_JOBS} > /dev/null 2>&1 && \
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

COPY --from=node-builder /root/.bun/bin/bun /usr/local/bin/bun
RUN ln -sf /usr/local/bin/bun /usr/local/bin/node && \
    ln -sf /usr/local/bin/bun /usr/local/bin/npm && \
    ln -sf /usr/local/bin/bun /usr/local/bin/npx
    
SHELL ["/usr/bin/zsh", "-o", "pipefail", "-c"]


FROM ${IMAGE}:${TAG} AS final

COPY --from=install / /

ENV PATH=/usr/local/go/bin:$PATH
ENV PIP_ROOT_USER_ACTION=ignore

CMD /usr/bin/zsh
