ARG IMAGE=ubuntu
ARG TAG=24.04

FROM ${IMAGE}:${TAG}

RUN apt-get update
RUN apt-get install -y curl
RUN apt-get install -y git
RUN apt-get install -y sudo
RUN apt-get install -y zsh

RUN curl -s https://dl.google.com/go/go1.23.6.linux-amd64.tar.gz | tar -C /usr/local -xz

SHELL ["/usr/bin/zsh", "-o", "pipefail", "-c"]

# Download and install nvm
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

CMD /usr/bin/zsh
