FROM golang:1.26.10-bookworm AS builder

# SECURITY: Do not pass credentials via `--build-arg` / `ARG` -- they are
# embedded in image metadata (`docker history`, OCI provenance) and leak to
# anyone with pull access. If a private Go module is ever needed, use a
# BuildKit secret (`RUN --mount=type=secret,id=...`). See DEVOP-586.

ENV GOTOOLCHAIN=local
ENV GOMODCACHE=/gocache

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN mkdir -p $GOMODCACHE

# Retries network-dependent commands with backoff, showing output only on failure.
# The project build runs outside it with GOPROXY=off, so a build error fails on its first attempt.
COPY --chmod=755 scripts/retry-fetch.sh /usr/local/bin/retry-fetch

# The module cache stays in the image. A cache mount is not part of the layer,
# so a later offline build misses the modules when that mount is empty.
RUN retry-fetch go mod download

ADD . /src

# Optional version stamped into the binary (not a secret). When unset, the
# Makefile derives it from git (tag, or <branch>-<commit>). Declared after the
# module download so a new VERSION does not invalidate that layer.
ARG VERSION

RUN --mount=type=cache,target=/root/.cache/go-build \
    GOPROXY=off make install

#==============================================================

FROM debian:bookworm-slim AS execution

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
