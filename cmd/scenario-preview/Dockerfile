# syntax=docker/dockerfile:1
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY embed.go ./
COPY internal ./internal
COPY cmd/scenario-preview ./cmd/scenario-preview
COPY web ./web
COPY scenarios ./scenarios
COPY vm ./vm
COPY config ./config
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/scenario-preview ./cmd/scenario-preview

FROM alpine:3.22
RUN apk add --no-cache tmux ttyd
COPY scenarios /scenarios
COPY --from=build /out/scenario-preview /usr/local/bin/scenario-preview
COPY cmd/scenario-preview/shims/ /opt/scenario-preview/bin/
COPY cmd/scenario-preview/entrypoint.sh /usr/local/bin/scenario-preview-entrypoint
RUN chmod 0555 /usr/local/bin/scenario-preview /usr/local/bin/scenario-preview-entrypoint /opt/scenario-preview/bin/*
EXPOSE 8080
ENTRYPOINT ["scenario-preview-entrypoint"]
