# Copyright 2017 The Codefresh Authors. All rights reserved.
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
#
# ----- Go Dev  Image ------
#
FROM golang:1.10 AS godev

# set working directory
RUN mkdir -p /go/src/github.com/codefresh-io/hermes
WORKDIR /go/src/github.com/codefresh-io/hermes

# copy sources
COPY . .

#
# ------ Go Test Runner ------
#
FROM godev AS tester

# run tests
RUN hack/test.sh

# upload coverage reports to Codecov.io: pass CODECOV_TOKEN as build-arg
ARG CODECOV_TOKEN
# default codecov bash uploader (sometimes it's worth to use GitHub version or custom one, to avoid bugs)
ARG CODECOV_BASH_URL=https://codecov.io/bash
# set Codecov expected env
ARG VCS_COMMIT_ID
ARG VCS_BRANCH_NAME
ARG VCS_SLUG
ARG CI_BUILD_URL
ARG CI_BUILD_ID
RUN if [ "$CODECOV_TOKEN" != "" ]; then curl -s $CODECOV_BASH_URL | bash -s; fi

#
# ------ Go Builder ------
#
FROM godev AS builder

# build binary
ARG VERSION
RUN make build

#
# ------ Hermes Trigger manager image ------
#
FROM alpine:3.9

RUN apk update && apk add --no-cache ca-certificates && apk upgrade

ENV GIN_MODE=release

COPY --from=builder /go/src/github.com/codefresh-io/hermes/.bin/hermes /usr/local/bin/hermes
COPY --from=builder /go/src/github.com/codefresh-io/hermes/pkg/backend/dev_compose_types.json /etc/hermes/dev_compose_types.json
COPY --from=builder /go/src/github.com/codefresh-io/hermes/pkg/backend/dev_external_types.json /etc/hermes/dev_external_types.json

ENTRYPOINT ["/usr/local/bin/hermes"]
CMD ["server"]

ARG VCS_COMMIT_ID
LABEL org.label-schema.vcs-ref=$VCS_COMMIT_ID \
      org.label-schema.vcs-url="https://github.com/codefresh-io/hermes" \
      org.label-schema.description="Hermes is a Codefresh trigger manager" \
      org.label-schema.vendor="Codefresh Inc." \
      org.label-schema.url="https://github.com/codefresh-io/hermes" \
      org.label-schema.docker.cmd="docker run -d --rm -p 80:8080 codefreshio/hermes server" \
      org.label-schema.docker.cmd.help="docker run -it --rm codefreshio/hermes --help"
