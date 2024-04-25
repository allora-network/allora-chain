FROM --platform=linux/amd64 golang:1.21-bookworm AS builder

ARG GH_TOKEN

ADD . /src
WORKDIR /src

RUN make install

#==============================================================

FROM debian:12.5-slim as execution

ENV DEBIAN_FRONTEND=noninteractive \
    USERNAME=appuser \
    APP_PATH=/data

#* curl jq - required for readyness probe and to download genesis
RUN apt update && \
    apt -y dist-upgrade && \
    apt install -y --no-install-recommends \
        curl jq \
        tzdata \
        ca-certificates && \
    echo "deb http://deb.debian.org/debian testing main" >> /etc/apt/sources.list && \
    apt update && \
    apt install -y --no-install-recommends -t testing \
      zlib1g \
      libgnutls30 \
      perl-base && \
    rm -rf /var/cache/apt/*

#* Install dasel to work with json/yaml/toml configs
ENV DASEL_VERSION="v2.6.0"
ADD https://github.com/TomWright/dasel/releases/download/${DASEL_VERSION}/dasel_linux_amd64 /usr/local/bin/dasel
RUN chmod a+x /usr/local/bin/dasel

COPY --from=builder /go/bin/* /usr/local/bin/

RUN groupadd -g 1001 ${USERNAME} \
    && useradd -m -d ${APP_PATH} -u 1001 -g 1001 ${USERNAME}

EXPOSE 26656 26657
VOLUME ${APP_PATH}
WORKDIR ${APP_PATH}

USER ${USERNAME}

ENTRYPOINT ["allorad"]
