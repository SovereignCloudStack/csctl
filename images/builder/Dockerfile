# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This is the Dockerfile for the Builder Image that is used by the CI
# pipelines.
# If you make changes to this Dockerfile run `make builder-image-push`.

# Install Lychee
FROM docker.io/library/alpine:3.23.4@sha256:4d889c14e7d5a73929ab00be2ef8ff22437e7cbc545931e52554a7b00e123d8b as lychee
# update: datasource=github-tags depName=lycheeverse/lychee versioning=semver
ENV LYCHEE_VERSION="v0.24.2"
# hadolint ignore=DL3018
RUN apk add --no-cache curl && \
  curl -L -o /tmp/lychee-${LYCHEE_VERSION}.tgz https://github.com/lycheeverse/lychee/releases/download/lychee-${LYCHEE_VERSION}/lychee-x86_64-unknown-linux-gnu.tar.gz && \
  tar -xz -C /tmp -f /tmp/lychee-${LYCHEE_VERSION}.tgz && \
  mv /tmp/lychee-x86_64-unknown-linux-gnu/lychee /usr/bin/lychee && \
  rm -rf /tmp/linux-amd64 /tmp/lychee-${LYCHEE_VERSION}.tgz

# Install Golang CI Lint
FROM docker.io/library/alpine:3.23.4@sha256:4d889c14e7d5a73929ab00be2ef8ff22437e7cbc545931e52554a7b00e123d8b as golangci
# update: datasource=github-tags depName=golangci/golangci-lint versioning=semver
ENV GOLANGCI_VERSION="v2.12.2"
WORKDIR /
# hadolint ignore=DL3018,DL4006
RUN apk add --no-cache curl && \
  curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/main/install.sh | \
  sh -s ${GOLANGCI_VERSION}

# Install Hadolint
FROM docker.io/hadolint/hadolint:v2.14.0-alpine@sha256:7aba693c1442eb31c0b015c129697cb3b6cb7da589d85c7562f9deb435a6657c as hadolint

# Install Trivy
FROM docker.io/aquasec/trivy:0.70.0@sha256:be1190afcb28352bfddc4ddeb71470835d16462af68d310f9f4bca710961a41e as trivy

############################
# csctl Build Image Base #
############################
FROM docker.io/library/golang:1.25-bookworm

# update: datasource=repology depName=debian_11/skopeo versioning=loose
ENV SKOPEO_VERSION="1.9.3+ds1-1+b10"
# update: datasource=github-tags depName=adrienverge/yamllint versioning=semver
ENV YAMLLINT_VERSION="v1.38.0"
# update: datasource=github-tags depName=opt-nc/yamlfixer versioning=semver
ENV YAMLFIXER_VERSION="0.9.15"

# hadolint ignore=DL3008
RUN apt-get update && \
  apt-get install -qy --no-install-recommends \
  gnupg python3 python3-pip \
  file zip unzip jq gettext \
  skopeo=${SKOPEO_VERSION} \
  protobuf-compiler libprotobuf-dev \
  libsystemd-dev jq && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
  pip install --break-system-packages --no-cache-dir \
  yamllint==${YAMLLINT_VERSION} \
  yamlfixer-opt-nc==${YAMLFIXER_VERSION}

COPY --from=lychee /usr/bin/lychee /usr/bin/lychee
COPY --from=golangci /bin/golangci-lint /usr/local/bin
COPY --from=hadolint /bin/hadolint /usr/bin/hadolint
COPY --from=trivy /usr/local/bin/trivy /usr/bin/trivy

ENV GOCACHE=/go/cache

COPY build.sh /
RUN chmod +x /build.sh
ENTRYPOINT ["/build.sh"]
