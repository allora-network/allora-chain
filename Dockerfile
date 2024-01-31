FROM --platform=linux/amd64 golang:1.21-bookworm AS builder

ARG GH_TOKEN

ADD . /src
WORKDIR /src

# Set up git for private repos
RUN git config --global url."https://${GH_TOKEN}@github.com".insteadOf "https://github.com"
ENV GOPRIVATE="github.com/upshot-tech/"
RUN make install

#==============================================================

FROM debian:bookworm-slim as execution

ENV DEBIAN_FRONTEND=noninteractive \
    USERNAME=appuser \
    APP_PATH=/data

RUN apt update && \
    apt -y dist-upgrade && \
    apt install -y --no-install-recommends \
        tzdata \
        ca-certificates && \
    rm -rf /var/cache/apt/*

COPY --from=builder /go/bin/* /usr/local/bin/
COPY scripts/init.sh /init.sh

RUN groupadd -g 1001 ${USERNAME} \
    && useradd -m -d ${APP_PATH} -u 1001 -g 1001 ${USERNAME}

EXPOSE 26657 1317
VOLUME ${APP_PATH}
WORKDIR ${APP_PATH}

USER ${USERNAME}

ENTRYPOINT ["uptd"]
