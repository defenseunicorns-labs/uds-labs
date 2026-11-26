# syntax=docker/dockerfile:1

ARG EDUCATES_VERSION=3.8.0
FROM ghcr.io/educates/educates-base-environment:${EDUCATES_VERSION}

ARG TARGETARCH
ARG UDS_VERSION=0.35.1

USER root

RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) uds_sha256="4ea7d553900199f36a4976bab526c780cac23a1025fb167e643d247682f252d4" ;; \
      arm64) uds_sha256="0915934b29bf551929388fae5e661094e3643d2ceed5548d4c7d300f612f9b8e" ;; \
      *) echo "unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl --fail --location --silent --show-error \
      --output /usr/local/bin/uds \
      "https://github.com/defenseunicorns/uds-cli/releases/download/v${UDS_VERSION}/uds-cli_v${UDS_VERSION}_Linux_${TARGETARCH}"; \
    echo "${uds_sha256}  /usr/local/bin/uds" | sha256sum --check --status; \
    chmod 0755 /usr/local/bin/uds

COPY --chown=1001:0 workshop /opt/workshop
COPY --chown=1001:0 exercises /home/eduk8s/exercises
COPY --chown=1001:0 README.md /home/eduk8s/README.md

RUN fix-permissions /home/eduk8s /opt/workshop

USER 1001
